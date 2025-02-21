package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/costinul/git-rest-cache/config"
	"github.com/costinul/git-rest-cache/gitcache"
	"github.com/costinul/git-rest-cache/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockProviderManager struct {
	defaultProvider provider.ProviderManager
}

type mockProvider struct {
	gitProvider provider.Provider
}

type mockProviderRepo struct {
	gitRepo provider.ProviderRepo
	repo    string
	token   string
}

func (m *mockProvider) GetURLPath() string {
	return "/github/:owner/:repo"
}

func newMockProviderManager() *mockProviderManager {
	return &mockProviderManager{
		defaultProvider: provider.NewDefaultProviderManager(),
	}
}

func (m *mockProviderManager) GetProviders() []provider.Provider {
	return []provider.Provider{
		&mockProvider{gitProvider: m.defaultProvider.GetProvider("github")},
	}
}

func (m *mockProviderManager) GetProvider(name string) provider.Provider {
	return nil
}

func (m *mockProvider) GetRepo(c *gin.Context) (provider.ProviderRepo, error) {
	token := c.GetHeader("X-Token")
	gitRepo, err := m.gitProvider.GetRepo(c)
	if err != nil {
		return nil, err
	}

	return &mockProviderRepo{
		gitRepo: gitRepo,
		repo:    c.Param("repo"),
		token:   token,
	}, nil
}

func (r *mockProviderRepo) Hash() string {
	return r.gitRepo.Hash()
}

func (r *mockProviderRepo) RepoURL() string {
	return r.gitRepo.RepoURL()
}

func (r *mockProviderRepo) GitURL() string {
	return r.gitRepo.GitURL()
}

func (m *mockProviderRepo) ValidateToken(token string) (bool, error) {
	if m.repo == "private-repo" {
		if token == "valid-token" {
			return true, nil
		}
		return false, nil
	}
	return true, nil
}

func readFile(gitUrl, branch, filePath string) ([]byte, error) {
	if filePath == "/notfound.txt" {
		return nil, gitcache.ErrFileNotFound
	}
	return []byte(fmt.Sprintf("content for url=%s, branch=%s, file=%s", gitUrl, branch, filePath)), nil
}

func TestAPIEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		StorageFolder: "storage/cache",
		Port:          8080,
	}

	ctx := context.Background()
	gitManager := gitcache.NewTestGitManager(readFile)

	gitCache := gitcache.NewGitCache(cfg, ctx, gitManager)
	err := gitCache.Start()
	if err != nil {
		t.Fatalf("Failed to start git cache: %v", err)
	}

	api := NewCacheAPI(cfg, gitCache, newMockProviderManager())
	router := api.Router()

	tests := []struct {
		name       string
		path       string
		method     string
		token      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Valid token for private repo",
			path:       "/github/test/private-repo/main/file.txt",
			method:     "GET",
			token:      "valid-token",
			wantStatus: http.StatusOK,
			wantBody:   "content for url=https://valid-token@github.com/test/private-repo.git, branch=main, file=/file.txt",
		},
		{
			name:       "Invalid token for private repo",
			path:       "/github/test/private-repo/main/file.txt",
			method:     "GET",
			token:      "invalid-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Public repo with missing token",
			path:       "/github/test/public-repo/main/file.txt",
			method:     "GET",
			token:      "",
			wantStatus: http.StatusOK,
			wantBody:   "content for url=https://github.com/test/public-repo.git, branch=main, file=/file.txt",
		},
		{
			name:       "Public repo with inexistent file",
			path:       "/github/test/public-repo/main/notfound.txt",
			method:     "GET",
			token:      "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Invalid provider",
			path:       "/invalid/test/repo/main/file.txt",
			method:     "GET",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(tt.method, tt.path, nil)
			assert.NoError(t, err)

			if tt.token != "" {
				req.Header.Set("X-Token", tt.token)
			}
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}
