package store

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/internal/bot/store.redis -o redis_mock_test.go -n RedisMock -p store

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

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

func TestStore_GetUsersStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("usage read error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.When(ctx, dataUsageKey).Then(nil, errors.New("boom"))

		s := New(r)
		stats, err := s.GetUsersStats(ctx)
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.ErrorContains(t, err, "failed to get usage data")
	})

	t.Run("auth date read error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.When(ctx, dataUsageKey).Then(map[string]string{}, nil)
		r.HGetAllMock.When(ctx, authDateKey).Then(nil, errors.New("boom"))

		s := New(r)
		stats, err := s.GetUsersStats(ctx)
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.ErrorContains(t, err, "failed to get auth date data")
	})

	t.Run("usage parse error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.When(ctx, dataUsageKey).Then(map[string]string{"u1": "not-int"}, nil)
		r.HGetAllMock.When(ctx, authDateKey).Then(map[string]string{}, nil)

		s := New(r)
		stats, err := s.GetUsersStats(ctx)
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.ErrorContains(t, err, "failed to parse usage for u1")
	})

	t.Run("stats are sorted and formatted", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.When(ctx, dataUsageKey).Then(map[string]string{
			"u1": "2048",
			"u2": "1",
		}, nil)
		r.HGetAllMock.When(ctx, authDateKey).Then(map[string]string{
			"u1": "",
			"u2": "bad-date",
		}, nil)

		s := New(r)
		stats, err := s.GetUsersStats(ctx)
		require.NoError(t, err)
		require.Len(t, stats, 2)

		assert.Equal(t, 1, stats[0].Num)
		assert.Equal(t, "u1", stats[0].Username)
		assert.Equal(t, int64(2048), stats[0].UsageRaw)
		assert.Equal(t, "2.00 KB", stats[0].Usage)
		assert.Equal(t, "-", stats[0].LastAuth)

		assert.Equal(t, 2, stats[1].Num)
		assert.Equal(t, "u2", stats[1].Username)
		assert.Equal(t, int64(1), stats[1].UsageRaw)
		assert.Equal(t, "1 B", stats[1].Usage)
		assert.Equal(t, "bad-date", stats[1].LastAuth)
	})
}

func TestStore_CreateUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("already exists", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("hash", nil)

		s := New(r)
		err := s.CreateUser(ctx, "alice", "secret")
		require.Error(t, err)
		assert.EqualError(t, err, "user with provided username already exists")
	})

	t.Run("exists check error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", errors.New("boom"))

		s := New(r)
		err := s.CreateUser(ctx, "alice", "secret")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if user exists")
	})

	t.Run("create error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", goredis.Nil)
		r.HSetMock.Return(errors.New("save failed"))

		s := New(r)
		err := s.CreateUser(ctx, "alice", "secret")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create user")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", goredis.Nil)
		r.HSetMock.Set(func(callCtx context.Context, key string, values ...interface{}) error {
			require.Equal(t, ctx, callCtx)
			require.Equal(t, userAuthKey, key)
			require.Len(t, values, 2)
			require.Equal(t, "alice", values[0])
			hash, ok := values[1].(string)
			require.True(t, ok)
			require.NotEmpty(t, hash)

			return nil
		})

		s := New(r)
		err := s.CreateUser(ctx, "alice", "secret")
		require.NoError(t, err)
	})
}

func TestStore_DeleteUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", goredis.Nil)

		s := New(r)
		err := s.DeleteUser(ctx, "alice")
		require.Error(t, err)
		assert.EqualError(t, err, "user with provided username not found")
	})

	t.Run("exists check error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", errors.New("boom"))

		s := New(r)
		err := s.DeleteUser(ctx, "alice")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if user exists")
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("hash", nil)
		r.HDelMock.Expect(ctx, userAuthKey, "alice").Return(errors.New("delete failed"))

		s := New(r)
		err := s.DeleteUser(ctx, "alice")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete user")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("hash", nil)
		r.HDelMock.Expect(ctx, userAuthKey, "alice").Return(nil)

		s := New(r)
		err := s.DeleteUser(ctx, "alice")
		require.NoError(t, err)
	})
}

func TestStore_IsUsernameFree(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("free", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", goredis.Nil)

		s := New(r)
		isFree, err := s.IsUsernameFree(ctx, "alice")
		require.NoError(t, err)
		assert.True(t, isFree)
	})

	t.Run("redis error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("", errors.New("boom"))

		s := New(r)
		isFree, err := s.IsUsernameFree(ctx, "alice")
		require.Error(t, err)
		assert.False(t, isFree)
		assert.ErrorContains(t, err, "failed to check if user exists")
	})

	t.Run("taken", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetMock.Expect(ctx, userAuthKey, "alice").Return("hash", nil)

		s := New(r)
		isFree, err := s.IsUsernameFree(ctx, "alice")
		require.NoError(t, err)
		assert.False(t, isFree)
	})
}

func TestStore_GetUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.Expect(ctx, userAuthKey).Return(nil, errors.New("boom"))

		s := New(r)
		users, err := s.GetUsers(ctx)
		require.Error(t, err)
		assert.Nil(t, users)
		assert.ErrorContains(t, err, "failed to get users")
	})

	t.Run("sorted users", func(t *testing.T) {
		t.Parallel()

		r := NewRedisMock(t)
		r.HGetAllMock.Expect(ctx, userAuthKey).Return(map[string]string{"bob": "", "alice": ""}, nil)

		s := New(r)
		users, err := s.GetUsers(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob"}, users)
		assert.True(t, sort.StringsAreSorted(users))
	})
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "bytes", size: 100, want: "100 B"},
		{name: "kb boundary", size: 1024, want: "1024 B"},
		{name: "kb", size: 1025, want: "1.00 KB"},
		{name: "mb", size: 2 * 1024 * 1024, want: "2.00 MB"},
		{name: "gb", size: 3 * 1024 * 1024 * 1024, want: "3.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatBytes(tt.size))
		})
	}
}

func TestFormatFromNow(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "-"},
		{name: "invalid date", raw: "bad", want: "bad"},
		{name: "just now", raw: now.Add(-20 * time.Second).Format(jsTimeLayout), want: "just now"},
		{name: "minutes", raw: now.Add(-5 * time.Minute).Format(jsTimeLayout), want: "5 minutes ago"},
		{name: "hours", raw: now.Add(-2 * time.Hour).Format(jsTimeLayout), want: "2 hours ago"},
		{name: "days", raw: now.Add(-49 * time.Hour).Format(jsTimeLayout), want: "2 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatFromNow(tt.raw))
		})
	}
}

func TestCleanPublicHost(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "example.com", CleanPublicHost("https://example.com/"))
	assert.Equal(t, "example.com", CleanPublicHost("http://example.com"))
	assert.Equal(t, "example.com:8443/path", CleanPublicHost("https://example.com:8443/path/"))
}
