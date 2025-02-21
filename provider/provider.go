package provider

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

type ProviderRepo interface {
	Hash() string
	RepoURL() string
	ValidateToken(token string) (bool, error)
	GitURL() string
}

type Provider interface {
	GetURLPath() string
	GetRepo(c *gin.Context) (ProviderRepo, error)
}

type ProviderManager interface {
	GetProviders() []Provider
	GetProvider(name string) Provider
}

type DefaultProviderManager struct {
	providers map[string]Provider
}

const (
	REPO_HASH_SALT = "d57bbdf3b5614008a74b20891834d223"
)

func NewDefaultProviderManager() *DefaultProviderManager {
	return &DefaultProviderManager{
		providers: map[string]Provider{
			"github": &githubProvider{},
		},
	}
}

func (pm *DefaultProviderManager) GetProviders() []Provider {
	providers := make([]Provider, 0, len(pm.providers))
	for _, provider := range pm.providers {
		providers = append(providers, provider)
	}
	return providers
}

func computeRepoHash(provider, owner, repo, token string) string {
	hash := sha1.New()

	hash.Write([]byte(provider))
	hash.Write([]byte(owner))
	hash.Write([]byte(repo))

	if token != "" {
		hash.Write([]byte(token))
	}

	hash.Write([]byte(REPO_HASH_SALT))

	fullHash := hash.Sum(nil)

	return hex.EncodeToString(fullHash)[:24]
}

func (pm *DefaultProviderManager) GetProvider(name string) Provider {
	return pm.providers[name]
}
