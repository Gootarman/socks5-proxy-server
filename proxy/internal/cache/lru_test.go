package cache

import (
	"reflect"
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestExpirableLRU_AddGet(t *testing.T) {
	t.Parallel()

	cache := NewExpirableLRU[string, int](2, time.Hour)
	stopExpirableLRUGC(cache)
	cache.Add("k1", 42)

	v, ok := cache.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestExpirableLRU_Expires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewExpirableLRU[string, string](2, 10*time.Millisecond)
		cache.Add("k1", "v1")
		time.Sleep(20 * time.Millisecond)

		v, ok := cache.Get("k1")
		assert.False(t, ok)
		assert.Equal(t, "", v)

		stopExpirableLRUGC(cache)
		synctest.Wait()
	})
}

func stopExpirableLRUGC[K comparable, V any](c *ExpirableLRU[K, V]) {
	doneField := reflect.ValueOf(c.cache).Elem().FieldByName("done")
	doneCh := *(*chan struct{})(unsafe.Pointer(doneField.UnsafeAddr()))
	close(doneCh)
}
