package gitcache

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type GitCacheManager interface {
	readFile(b *gitBranch, filePath string) ([]byte, error)
	cloneBranch(b *gitBranch) error
	updateBranch(b *gitBranch) error
	deleteBranch(b *gitBranch) error
	containsBranch(b *gitBranch) bool
	deleteRepo(r *gitRepo) error
	getCachedRepoBranches(storageFolder string) ([]repoBranchInfo, error)
	listTree(b *gitBranch, path string) ([]byte, error)
}

type DefaultGitManager struct{}

type TestGitManager struct {
	ReadFileCallback func(gitUrl, branch, filePath string) ([]byte, error)
	ListTreeCallback func(gitUrl, branch, path string) ([]byte, error)
}

func (m *DefaultGitManager) readFile(b *gitBranch, filePath string) ([]byte, error) {
	b.repo.rmu.RLock()
	defer b.repo.rmu.RUnlock()

	fp := filepath.Join(b.path, filepath.FromSlash(filePath))
	content, err := os.ReadFile(fp)
	if os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

func (m *DefaultGitManager) cloneBranch(b *gitBranch) error {
	err := m.runCommand(b.repo.cache.ctx, "git", "clone", "--depth=1", "--branch", b.name, b.repo.gitUrl, b.path)
	if err != nil {
		return fmt.Errorf("failed to clone branch: %w", err)
	}

	return nil
}

func (m *DefaultGitManager) updateBranch(b *gitBranch) error {
	cmd := exec.CommandContext(b.repo.cache.ctx, "git", "-C", b.path, "fetch", "origin", b.name, "--depth=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch branch: %w, output: %s", err, string(output))
	}

	cmd = exec.CommandContext(b.repo.cache.ctx, "git", "-C", b.path, "reset", "--hard", "origin/"+b.name)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reset branch: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *DefaultGitManager) deleteBranch(b *gitBranch) error {
	err := os.RemoveAll(b.path)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

func (m *DefaultGitManager) deleteRepo(r *gitRepo) error {
	files, err := os.ReadDir(r.path)
	if err != nil {
		return fmt.Errorf("failed to read repo directory: %w", err)
	}

	if len(files) == 0 {
		err = os.RemoveAll(r.path)
		if err != nil {
			return fmt.Errorf("failed to delete repo: %w", err)
		}
	}

	return nil
}

func (m *DefaultGitManager) containsBranch(b *gitBranch) bool {
	info, err := os.Stat(b.path)
	if os.IsNotExist(err) {
		return false
	}

	if !info.IsDir() {
		return false
	}

	gitPath := filepath.Join(b.path, ".git")
	gitInfo, err := os.Stat(gitPath)
	if os.IsNotExist(err) {
		return false
	}

	return gitInfo.IsDir()
}

func (e *DefaultGitManager) runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run command: %w, output: %s", err, string(output))
	}
	return nil
}

func (m *DefaultGitManager) getCachedRepoBranches(storageFolder string) ([]repoBranchInfo, error) {
	list := []repoBranchInfo{}
	repos, err := os.ReadDir(storageFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage folder: %w", err)
	}

	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}

		hash := repo.Name()
		repoPath := filepath.Join(storageFolder, hash)
		branches, err := os.ReadDir(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read repo folder %s: %w", repoPath, err)
		}

		for _, branch := range branches {
			if !branch.IsDir() {
				continue
			}

			branchName := branch.Name()
			branchPath := filepath.Join(repoPath, branchName)
			gitUrl, err := getGitURL(branchPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get git url for branch %s: %w", branchName, err)
			}

			list = append(list, repoBranchInfo{hash, branchName, gitUrl})
		}

	}

	return list, nil
}

func (m *DefaultGitManager) listTree(b *gitBranch, path string) ([]byte, error) {
	fullPath := filepath.Join(b.path, path)
	if info, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
	} else if !info.IsDir() {
		return nil, ErrFileNotFound
	}

	cmd := exec.CommandContext(b.repo.cache.ctx, "git", "-C", b.path, "ls-tree", "-l", "HEAD:"+path)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("failed to list tree: %w, output: %s", err, string(output))
	}

	return output, nil
}

// TestGitManager
func NewTestGitManager(readFileCallback func(gitUrl, branch, filePath string) ([]byte, error),
	listTreeCallback func(gitUrl, branch, path string) ([]byte, error)) *TestGitManager {
	return &TestGitManager{ReadFileCallback: readFileCallback, ListTreeCallback: listTreeCallback}
}

func (m *TestGitManager) readFile(b *gitBranch, filePath string) ([]byte, error) {
	content, err := m.ReadFileCallback(b.repo.gitUrl, b.name, filePath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (m *TestGitManager) cloneBranch(b *gitBranch) error {
	return nil
}

func (m *TestGitManager) updateBranch(b *gitBranch) error {
	return nil
}

func (m *TestGitManager) deleteBranch(b *gitBranch) error {
	return nil
}

func (m *TestGitManager) deleteRepo(r *gitRepo) error {
	return nil
}

func (m *TestGitManager) containsBranch(b *gitBranch) bool {
	return true
}

func (e *TestGitManager) runCommand(ctx context.Context, name string, args ...string) error {
	return nil
}

func (m *TestGitManager) getCachedRepoBranches(storageFolder string) ([]repoBranchInfo, error) {
	return []repoBranchInfo{}, nil
}

func (m *TestGitManager) listTree(b *gitBranch, path string) ([]byte, error) {
	content, err := m.ListTreeCallback(b.repo.gitUrl, b.name, path)
	if err != nil {
		return nil, err
	}
	return content, nil
}
