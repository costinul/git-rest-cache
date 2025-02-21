package provider

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type githubProvider struct{}

type githubRepo struct {
	owner string
	repo  string
	token string
}

func (p *githubProvider) GetURLPath() string {
	return "/github/:owner/:repo"
}

func (p *githubProvider) GetRepo(c *gin.Context) (ProviderRepo, error) {
	token := c.GetHeader("X-Token")

	return &githubRepo{
		owner: c.Param("owner"),
		repo:  c.Param("repo"),
		token: token,
	}, nil
}

func (r *githubRepo) Hash() string {
	return computeRepoHash("github", r.owner, r.repo, r.token)
}

func (r *githubRepo) RepoURL() string {
	return fmt.Sprintf("https://github.com/%s/%s", r.owner, r.repo)
}

func (r *githubRepo) ValidateToken(token string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", r.owner, r.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", "git-rest-cache/1.0")

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (r *githubRepo) GitURL() string {
	if r.token != "" {
		return fmt.Sprintf("https://%s@github.com/%s/%s.git", r.token, r.owner, r.repo)
	}

	return fmt.Sprintf("https://github.com/%s/%s.git", r.owner, r.repo)
}
