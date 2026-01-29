package cache

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type ExpirableLRU[K comparable, V any] struct {
	cache *expirable.LRU[K, V]
}

func NewExpirableLRU[K comparable, V any](maxSize int, ttl time.Duration) *ExpirableLRU[K, V] {
	return &ExpirableLRU[K, V]{
		cache: expirable.NewLRU[K, V](maxSize, nil, ttl),
	}
}

func (c *ExpirableLRU[K, V]) Add(key K, value V) {
	c.cache.Add(key, value)
}

func (c *ExpirableLRU[K, V]) Get(key K) (value V, exists bool) {
	return c.cache.Get(key)
}
