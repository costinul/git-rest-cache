package api

import (
	"fmt"
	"net/http"

	"github.com/costinul/git-rest-cache/config"
	"github.com/costinul/git-rest-cache/gitcache"
	"github.com/costinul/git-rest-cache/provider"
	"github.com/gin-gonic/gin"
)

type CacheAPI struct {
	gitCache *gitcache.GitCache
	gin      *gin.Engine
	cfg      *config.Config
}

func NewCacheAPI(cfg *config.Config, gitCache *gitcache.GitCache, providerManager provider.ProviderManager) *CacheAPI {
	router := gin.Default()

	providers := providerManager.GetProviders()
	for _, provider := range providers {
		urlPath := fmt.Sprintf("%v/:branch/*filepath", provider.GetURLPath())
		router.GET(urlPath, func(c *gin.Context) {
			getGitContent(c, gitCache, provider)
		})
	}

	api := CacheAPI{
		gin:      router,
		gitCache: gitCache,
		cfg:      cfg,
	}

	return &api
}

func (api *CacheAPI) Run() error {
	return api.gin.Run(fmt.Sprintf(":%d", api.cfg.Port))
}

func (api *CacheAPI) Router() *gin.Engine {
	return api.gin
}

func getGitContent(c *gin.Context, gitCache *gitcache.GitCache, provider provider.Provider) {
	token := c.GetHeader("X-Token")
	repo, err := provider.GetRepo(c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting repo")
		return
	}

	hasAccess, err := hasAccess(token, gitCache, repo)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if !hasAccess {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	data, err := gitCache.GetFileContent(repo.Hash(), repo.GitURL(), c.Param("branch"), c.Param("filepath"))
	if err != nil {
		if err == gitcache.ErrFileNotFound {
			c.String(http.StatusNotFound, "File not found")
		} else {
			c.String(http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", data)
}

func hasAccess(token string, gitCache *gitcache.GitCache, repo provider.ProviderRepo) (bool, error) {
	repoHash := repo.Hash()

	if !gitCache.HasAccess(token, repoHash) {
		validToken, err := repo.ValidateToken(token)
		if err != nil {
			return false, err
		}
		if validToken {
			gitCache.SetAccess(token, repoHash)
			return true, nil
		}
		return false, nil
	}

	return true, nil
}
