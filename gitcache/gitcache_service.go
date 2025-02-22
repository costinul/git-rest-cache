package gitcache

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/costinul/git-rest-cache/logger"
)

func (c *GitCache) Start() error {
	if err := c.verifySettings(); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	started := make(chan struct{})
	go func() {
		c.setRunning(true)
		close(started)
		if err := c.startRepoCheck(); err != nil {
			if err != context.Canceled {
				logger.Error(fmt.Sprintf("failed to start repo check: %v", err))
			}
		}
		c.setRunning(false)
	}()

	<-started

	return nil
}

func (c *GitCache) Stop() {
	if !c.IsRunning() {
		return
	}
	c.ctx.Done()
	c.tokenCache.Stop()
}

func (c *GitCache) setRunning(running bool) {
	c.cmu.Lock()
	c.running = running
	c.cmu.Unlock()
}

func (c *GitCache) IsRunning() bool {
	c.cmu.RLock()
	defer c.cmu.RUnlock()
	return c.running
}

func (c *GitCache) startRepoCheck() error {
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		if err := c.checkRepos(); err != nil {
			if err != context.Canceled {
				logger.Error(fmt.Sprintf("failed to check repos: %v", err))
			}
			time.Sleep(1 * time.Second)
			continue
		}

		select {
		case <-c.ctx.Done():
			if c.ctx.Err() != context.Canceled {
				return c.ctx.Err()
			}
			return nil
		case <-time.After(c.cfg.RepoCheckInterval):
		}
	}
}

func (c *GitCache) checkRepos() error {
	branches, err := c.manager.getCachedRepoBranches(c.cfg.StorageFolder)
	if err != nil {
		return fmt.Errorf("failed to get cached repo branches: %w", err)
	}

	for _, branch := range branches {
		b, err := c.getBranch(branch.hash, branch.gitUrl, branch.branch)
		if err != nil {
			return fmt.Errorf("failed to get repo from cache: %w", err)
		}

		if c.cfg.RepoTTL > 0 && b.isExpired() {
			err = b.delete()
			if err != nil {
				return fmt.Errorf("failed to delete repo: %w", err)
			}
			continue
		}

		err = b.update()
		if err != nil {
			return fmt.Errorf("failed to update repo: %w", err)
		}
	}

	return nil
}

func getGitURL(path string) (string, error) {
	out, err := exec.Command("git", "-C", path, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *GitCache) verifySettings() error {
	if _, err := os.Stat(c.cfg.StorageFolder); os.IsNotExist(err) {
		return fmt.Errorf("storage folder does not exist: %s", c.cfg.StorageFolder)
	}

	if c.cfg.RepoCheckInterval > c.cfg.RepoTTL/4 {
		return fmt.Errorf("repo check interval (%s) should be at most 1/4 of repo TTL (%s) for efficient cleanup",
			c.cfg.RepoCheckInterval, c.cfg.RepoTTL)
	}

	return nil
}
