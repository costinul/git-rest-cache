package api

import (
	"fmt"

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
	for _, p := range providers {
		blobPath := fmt.Sprintf("%v/:branch/blob/*filepath", p.GetURLPath())
		router.GET(blobPath, authMiddleware(gitCache, p), getGitBlobHandler(gitCache))

		listPath := fmt.Sprintf("%v/:branch/list/*path", p.GetURLPath())
		router.GET(listPath, authMiddleware(gitCache, p), getGitListHandler(gitCache))
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
