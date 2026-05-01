package cache

type NoCache struct{}

func NewNoCache() *NoCache {
	return &NoCache{}
}

func (c *NoCache) Get(chatID int64) ([]byte, error) {
	return nil, ErrCacheMiss
}
func (c *NoCache) Set(chatID int64, data []byte) error {
	return nil
}
func (c *NoCache) Delete(chatID int64) error {
	return nil
}
