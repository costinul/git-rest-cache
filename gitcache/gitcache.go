package gitcache

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/costinul/git-rest-cache/config"
	"github.com/costinul/git-rest-cache/logger"
	"github.com/karlseguin/ccache"
)

var ErrFileNotFound = fmt.Errorf("file not found")

type GitCache struct {
	cfg        *config.Config
	tokenCache *ccache.Cache
	repos      map[string]*gitRepo
	manager    GitCacheManager

	running bool
	ctx     context.Context
	cmu     sync.RWMutex
}

type gitRepo struct {
	cache    *GitCache
	hash     string
	gitUrl   string
	path     string
	branches map[string]*gitBranch

	rmu sync.RWMutex
}

type gitBranch struct {
	repo         *gitRepo
	name         string
	path         string
	cached       bool
	lastAccessed time.Time
}

type repoBranchInfo struct {
	hash   string
	branch string
	gitUrl string
}

func NewGitCache(cfg *config.Config, ctx context.Context, manager GitCacheManager) *GitCache {
	return &GitCache{
		cfg:        cfg,
		tokenCache: ccache.New(ccache.Configure().MaxSize(10000000)),
		repos:      make(map[string]*gitRepo),
		manager:    manager,
		ctx:        ctx,
	}
}

func (c *GitCache) GetFileContent(hash, gitUrl, branch, filePath string) ([]byte, error) {
	b, err := c.getBranch(hash, gitUrl, branch)
	if err != nil {
		return nil, err
	}
	content, err := b.readFile(filePath)
	if err != nil {
		return nil, err
	}

	b.touch()

	return content, nil
}

func (c *GitCache) getRepo(hash, gitUrl string) (*gitRepo, error) {
	c.cmu.RLock()
	r, ok := c.repos[hash]
	c.cmu.RUnlock()

	if !ok {
		c.cmu.Lock()
		defer c.cmu.Unlock()

		r = c.newRepo(c, hash, gitUrl)
		c.repos[hash] = r
	}

	return r, nil
}

func (c *GitCache) getBranch(hash, gitUrl, branch string) (*gitBranch, error) {
	repo, err := c.getRepo(hash, gitUrl)
	if err != nil {
		return nil, err
	}

	return repo.getBranch(branch)
}

func (c *GitCache) newRepo(cache *GitCache, hash, gitUrl string) *gitRepo {
	return &gitRepo{
		cache:    cache,
		hash:     hash,
		gitUrl:   gitUrl,
		path:     path.Join(cache.cfg.StorageFolder, hash),
		branches: make(map[string]*gitBranch),
	}
}

func (r *gitRepo) getBranch(branch string) (*gitBranch, error) {
	r.rmu.RLock()
	b, ok := r.branches[branch]
	r.rmu.RUnlock()

	if !ok {
		r.rmu.Lock()
		defer r.rmu.Unlock()

		b = r.newBranch(branch)
		r.branches[branch] = b
	}

	return b, nil
}

func (r *gitRepo) delete() error {
	r.rmu.Lock()
	defer r.rmu.Unlock()

	err := r.cache.manager.deleteRepo(r)
	if err != nil {
		return fmt.Errorf("failed to delete repo: %w", err)
	}

	r.cache.cmu.Lock()
	defer r.cache.cmu.Unlock()

	delete(r.cache.repos, r.hash)

	return nil
}

func (r *gitRepo) newBranch(branch string) *gitBranch {
	return &gitBranch{
		repo:         r,
		name:         branch,
		path:         path.Join(r.path, branch),
		cached:       false,
		lastAccessed: time.Now(),
	}
}

func (b *gitBranch) isCached() bool {
	b.repo.rmu.RLock()
	defer b.repo.rmu.RUnlock()

	if b.cached {
		return true
	}

	return b.repo.cache.manager.containsBranch(b)
}

func (b *gitBranch) touch() {
	b.repo.rmu.Lock()
	defer b.repo.rmu.Unlock()

	b.lastAccessed = time.Now()
}

func (b *gitBranch) isExpired() bool {
	b.repo.rmu.RLock()
	defer b.repo.rmu.RUnlock()

	return b.lastAccessed.Before(time.Now().Add(-b.repo.cache.cfg.RepoTTL))
}

func (b *gitBranch) cache() error {
	if b.isCached() {
		return nil
	}

	b.repo.rmu.Lock()
	defer b.repo.rmu.Unlock()

	err := b.repo.cache.manager.cloneBranch(b)
	if err != nil {
		return fmt.Errorf("failed to clone branch: %w", err)
	}

	b.cached = true

	return nil
}

func (b *gitBranch) update() error {
	if !b.isCached() {
		return nil
	}

	b.repo.rmu.Lock()
	defer b.repo.rmu.Unlock()

	err := b.repo.cache.manager.updateBranch(b)
	if err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	return nil
}

func (b *gitBranch) delete() error {
	if !b.isCached() {
		return nil
	}

	b.repo.rmu.Lock()
	defer func() {
		b.repo.rmu.Unlock()
		err := b.repo.delete()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to delete repo: %v", err))
		}
	}()

	err := b.repo.cache.manager.deleteBranch(b)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	delete(b.repo.branches, b.name)

	return nil
}

func (b *gitBranch) readFile(filePath string) ([]byte, error) {
	if err := b.cache(); err != nil {
		return nil, err
	}

	return b.repo.cache.manager.readFile(b, filePath)
}
