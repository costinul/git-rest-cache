package gitcache

func (c *GitCache) SetAccess(token, repoHash string) {
	key := buildKey(token, repoHash)
	c.tokenCache.Set(key, true, c.cfg.TokenTTL)
}

func (c *GitCache) HasAccess(token, repoHash string) bool {
	key := buildKey(token, repoHash)
	item := c.tokenCache.Get(key)
	if item == nil {
		return false
	}

	item.Extend(c.cfg.TokenTTL)
	return true
}

func (c *GitCache) RemoveAccess(token, repoHash string) {
	key := buildKey(token, repoHash)
	c.tokenCache.Delete(key)
}

func buildKey(token, repoHash string) string {
	return token + "|" + repoHash
}
