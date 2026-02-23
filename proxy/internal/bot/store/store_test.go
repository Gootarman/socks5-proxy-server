package store

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store.redis -o redis_mock_test.go -n RedisMock -p store

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goredis "github.com/redis/go-redis/v9"
)

func TestNew(t *testing.T) {
	t.Parallel()

	r := NewRedisMock(t)
	require.NotNil(t, New(r))
}

func TestStore_GetUserState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not found returns nil,nil", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userStateKey, "admin").Return("", goredis.Nil)

		s := New(r)
		state, err := s.GetUserState(ctx, "admin")
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("redis error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userStateKey, "admin").Return("", errors.New("boom"))

		s := New(r)
		state, err := s.GetUserState(ctx, "admin")
		require.Error(t, err)
		assert.Nil(t, state)
		assert.ErrorContains(t, err, "failed to get user state")
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userStateKey, "admin").Return("{", nil)

		s := New(r)
		state, err := s.GetUserState(ctx, "admin")
		require.Error(t, err)
		assert.Nil(t, state)
		assert.ErrorContains(t, err, "failed to unmarshal user state")
	})

	t.Run("nil data map gets normalized", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userStateKey, "admin").Return(`{"state":"idle"}`, nil)

		s := New(r)
		state, err := s.GetUserState(ctx, "admin")
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, StateIdle, state.State)
		require.NotNil(t, state.Data)
		assert.Empty(t, state.Data)
	})
}

func TestStore_SetUserState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("save state", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HSetMock.Set(func(callCtx context.Context, key string, values ...interface{}) error {
			require.Equal(t, ctx, callCtx)
			require.Equal(t, userStateKey, key)
			require.Len(t, values, 2)
			require.Equal(t, "admin", values[0])
			stateJSON, ok := values[1].([]byte)
			require.True(t, ok)
			assert.JSONEq(t, `{"state":"idle","data":{}}`, string(stateJSON))

			return nil
		})

		s := New(r)
		err := s.SetUserState(ctx, "admin", UserState{State: StateIdle})
		require.NoError(t, err)
	})

	t.Run("redis save error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HSetMock.Return(errors.New("save failed"))

		s := New(r)
		err := s.SetUserState(ctx, "admin", UserState{State: StateIdle})
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to save user state")
	})
}

func TestCleanPublicHost(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "example.com", CleanPublicHost("https://example.com/"))
	assert.Equal(t, "example.com", CleanPublicHost("http://example.com"))
	assert.Equal(t, "example.com:8443/path", CleanPublicHost("https://example.com:8443/path/"))
}
