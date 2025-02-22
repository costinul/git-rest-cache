package api

import (
	"net/http"

	"github.com/costinul/git-rest-cache/gitcache"
	"github.com/costinul/git-rest-cache/provider"
	"github.com/gin-gonic/gin"
)

func authMiddleware(gitCache *gitcache.GitCache, provider provider.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Token")
		repo, err := provider.GetRepo(c)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting repo")
			c.Abort()
			return
		}

		hasAccess, err := hasAccess(token, gitCache, repo)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			c.Abort()
			return
		}

		if !hasAccess {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		c.Set("repo", repo)
		c.Next()
	}
}

func getGitContentHandler(gitCache *gitcache.GitCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		repo, exists := c.Get("repo")
		if !exists {
			c.String(http.StatusInternalServerError, "Repo not found in context")
			return
		}

		providerRepo, ok := repo.(provider.ProviderRepo)
		if !ok {
			c.String(http.StatusInternalServerError, "Invalid repo type in context")
			return
		}

		data, err := gitCache.GetFileContent(providerRepo.Hash(), providerRepo.GitURL(), c.Param("branch"), c.Param("filepath"))
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
}
