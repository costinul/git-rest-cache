package gitcache

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/costinul/git-rest-cache/config"
	"github.com/stretchr/testify/assert"
)

type mockFileContent struct {
	hash    string
	branch  string
	gitUrl  string
	content string
}

type mockGitManager struct {
	mu       sync.RWMutex
	contents map[string]*mockFileContent
	getCount int
}

func newMockGitManager() *mockGitManager {
	return &mockGitManager{
		contents: make(map[string]*mockFileContent),
	}
}

func (m *mockGitManager) randomSleep(min, max time.Duration) {
	sleep := min + time.Duration(rand.Int63n(int64(max-min)))
	time.Sleep(sleep)
}

func (m *mockGitManager) contentKey(hash, branch string) string {
	return fmt.Sprintf("%s:%s", hash, branch)
}

func (m *mockGitManager) addContent(hash, branch, gitUrl, content string) {
	m.contents[m.contentKey(hash, branch)] = &mockFileContent{hash: hash, branch: branch, gitUrl: gitUrl, content: content}
}

func (m *mockGitManager) getContent(hash, branch string) (*mockFileContent, error) {
	content, exists := m.contents[m.contentKey(hash, branch)]
	if !exists {
		return nil, ErrFileNotFound
	}
	return content, nil
}

func (m *mockGitManager) cloneBranch(branch *gitBranch) error {
	m.randomSleep(100*time.Millisecond, 500*time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()

	content := fmt.Sprintf("Initial content for %s/%s", branch.repo.gitUrl, branch.name)
	m.addContent(branch.repo.hash, branch.name, branch.repo.gitUrl, content)
	return nil
}

func (m *mockGitManager) readFile(branch *gitBranch, filePath string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.randomSleep(10*time.Millisecond, 50*time.Millisecond)
	m.getCount++ // Increment counter
	content, err := m.getContent(branch.repo.hash, branch.name)
	if err != nil {
		return nil, err
	}
	return []byte(content.content), nil
}

func (m *mockGitManager) containsBranch(branch *gitBranch) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.contents[m.contentKey(branch.repo.hash, branch.name)]
	return exists
}

func (m *mockGitManager) deleteBranch(branch *gitBranch) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.contentKey(branch.repo.hash, branch.name)
	delete(m.contents, key)
	return nil
}

func (m *mockGitManager) deleteRepo(repo *gitRepo) error {
	return nil
}

func mockRepoHash(url string) string {
	return fmt.Sprintf("mock-hash-%s", url)
}

func (m *mockGitManager) getCachedRepoBranches(repoPath string) ([]repoBranchInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	branches := make([]repoBranchInfo, 0)

	for _, info := range m.contents {
		branches = append(branches, repoBranchInfo{
			hash:   info.hash,
			branch: info.branch,
			gitUrl: info.gitUrl,
		})
	}
	return branches, nil
}

func (m *mockGitManager) updateBranch(branch *gitBranch) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.randomSleep(50*time.Millisecond, 100*time.Millisecond)

	m.addContent(branch.repo.hash, branch.name, branch.repo.gitUrl, "Updated content")
	return nil
}

func (m *mockGitManager) getCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getCount
}

func TestGitCacheBasicFlow(t *testing.T) {
	cfg := &config.Config{
		StorageFolder:     "storage/cache",
		RepoTTL:           2 * time.Second,
		TokenTTL:          2 * time.Second,
		RepoCheckInterval: 1 * time.Second,
	}

	ctx := context.Background()
	mockManager := newMockGitManager()
	cache := NewGitCache(cfg, ctx, mockManager)

	err := cache.Start()
	assert.NoError(t, err, "Failed to start cache")
	defer cache.Stop()

	gitUrl := "https://github.com/test/repo"
	branch := "main"
	filePath := "test.txt"
	repoHash := mockRepoHash(gitUrl)

	content1, err := cache.GetFileContent(repoHash, gitUrl, branch, filePath)
	assert.NoError(t, err, "Failed initial GetFile")

	expectedInitial := fmt.Sprintf("Initial content for %s/%s", gitUrl, branch)
	assert.Equal(t, expectedInitial, string(content1))

	time.Sleep(2000 * time.Millisecond)

	content2, err := cache.GetFileContent(repoHash, gitUrl, branch, filePath)
	assert.NoError(t, err, "Failed second GetFile")
	assert.Equal(t, "Updated content", string(content2))

	time.Sleep(2500 * time.Millisecond)

	testBranch := &gitBranch{
		name: branch,
		repo: &gitRepo{
			gitUrl: gitUrl,
		},
	}
	assert.False(t, mockManager.containsBranch(testBranch), "Branch should be automatically removed after TTL")
}

func TestGitCacheConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		StorageFolder:     "storage/cache",
		RepoTTL:           5 * time.Second,
		TokenTTL:          5 * time.Second,
		RepoCheckInterval: 10 * time.Millisecond,
	}

	ctx := context.Background()
	mockManager := newMockGitManager()
	cache := NewGitCache(cfg, ctx, mockManager)
	err := cache.Start()
	assert.NoError(t, err, "Failed to start cache")
	defer cache.Stop()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	numGoroutines := 10
	iterationsPerGoroutine := 100
	expectedCalls := numGoroutines * iterationsPerGoroutine

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gitUrl := fmt.Sprintf("https://github.com/test/repo%d", i%3)
			branch := "main"
			filePath := "test.txt"

			for j := 0; j < iterationsPerGoroutine; j++ {
				repoHash := mockRepoHash(gitUrl)
				content, err := cache.GetFileContent(repoHash, gitUrl, branch, filePath)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: %v", i, j, err)
					return
				}
				assert.NotEmpty(t, content, "Content should not be empty")
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		assert.NoError(t, err, "Concurrent access error")
	}

	assert.Equal(t, expectedCalls, mockManager.getCallCount(), "Unexpected number of GetFileContent calls")
}

func TestGitCacheContextCancellation(t *testing.T) {
	cfg := &config.Config{
		StorageFolder:     "storage/cache",
		RepoTTL:           5 * time.Second,
		TokenTTL:          5 * time.Second,
		RepoCheckInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	mockManager := newMockGitManager()
	cache := NewGitCache(cfg, ctx, mockManager)

	err := cache.Start()
	assert.NoError(t, err, "Failed to start cache")
	assert.True(t, cache.IsRunning(), "Cache should be running after Start")

	cancel()

	deadline := time.After(5 * time.Second)
	for cache.IsRunning() {
		select {
		case <-deadline:
			assert.Fail(t, "Timeout waiting for cache to stop")
			return
		case <-time.After(100 * time.Millisecond):
		}
	}

	assert.False(t, cache.IsRunning(), "Cache should not be running after context cancellation")
}
