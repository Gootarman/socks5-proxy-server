package users

import (
	"context"
	"errors"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var (
			gotKey    string
			gotValues []interface{}
		)

		u := New(&redisStub{
			hgetFn: func(_ context.Context, key, field string) (string, error) {
				if key != userAuthKey || field != "alice" {
					t.Fatalf("unexpected HGet args: key=%q field=%q", key, field)
				}

				return "", goredis.Nil
			},
			hsetFn: func(_ context.Context, key string, values ...interface{}) error {
				gotKey = key
				gotValues = values

				return nil
			},
		})

		err := u.Create(context.Background(), "alice", "secret")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if gotKey != userAuthKey {
			t.Fatalf("expected HSet key %q, got %q", userAuthKey, gotKey)
		}

		if len(gotValues) != 2 {
			t.Fatalf("expected two HSet values, got %d", len(gotValues))
		}

		if gotValues[0] != "alice" {
			t.Fatalf("expected username alice, got %v", gotValues[0])
		}

		hash, ok := gotValues[1].(string)
		if !ok {
			t.Fatalf("expected hash string, got %T", gotValues[1])
		}

		if err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret")); err != nil {
			t.Fatalf("password is not hashed correctly: %v", err)
		}
	})

	t.Run("user exists", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetFn: func(_ context.Context, _, _ string) (string, error) {
			return "already-hashed", nil
		}})

		err := u.Create(context.Background(), "alice", "secret")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrUserExists) {
			t.Fatalf("expected ErrUserExists, got %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var (
			gotKey    string
			gotFields []string
		)

		u := New(&redisStub{
			hgetFn: func(_ context.Context, _, _ string) (string, error) {
				return "hash", nil
			},
			hdelFn: func(_ context.Context, key string, fields ...string) error {
				gotKey = key
				gotFields = fields

				return nil
			},
		})

		err := u.Delete(context.Background(), "alice")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if gotKey != userAuthKey {
			t.Fatalf("expected HDel key %q, got %q", userAuthKey, gotKey)
		}

		if len(gotFields) != 1 || gotFields[0] != "alice" {
			t.Fatalf("expected HDel fields [alice], got %v", gotFields)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetFn: func(_ context.Context, _, _ string) (string, error) {
			return "", goredis.Nil
		}})

		err := u.Delete(context.Background(), "alice")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestGetStats(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetAllFn: func(_ context.Context, key string) (map[string]string, error) {
			switch key {
			case userUsageDataKey:
				return map[string]string{
					"alice": "1025",
					"bob":   "256",
				}, nil
			case userAuthDateKey:
				return map[string]string{
					"alice": "2026-01-01T10:00:00Z",
				}, nil
			default:
				return nil, errors.New("unexpected key")
			}
		}})

		stats, err := u.GetStats(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(stats) != 2 {
			t.Fatalf("expected 2 stats, got %d", len(stats))
		}

		if stats[0].Username != "alice" || stats[1].Username != "bob" {
			t.Fatalf("expected sorted stats [alice bob], got [%s %s]", stats[0].Username, stats[1].Username)
		}

		if stats[0].Usage != "1.00 KB" {
			t.Fatalf("expected formatted usage 1.00 KB, got %q", stats[0].Usage)
		}

		if stats[0].LastAuth != "2026-01-01T10:00:00Z" {
			t.Fatalf("expected alice auth date, got %q", stats[0].LastAuth)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetAllFn: func(_ context.Context, key string) (map[string]string, error) {
			switch key {
			case userUsageDataKey:
				return map[string]string{"alice": "bad-int"}, nil
			case userAuthDateKey:
				return map[string]string{}, nil
			default:
				return nil, errors.New("unexpected key")
			}
		}})

		_, err := u.GetStats(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestIsUsernameFree(t *testing.T) {
	t.Parallel()

	t.Run("free", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetFn: func(_ context.Context, key, field string) (string, error) {
			if key != userAuthKey || field != "alice" {
				t.Fatalf("unexpected HGet args: key=%q field=%q", key, field)
			}

			return "", goredis.Nil
		}})

		isFree, err := u.IsUsernameFree(context.Background(), "alice")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !isFree {
			t.Fatal("expected username to be free")
		}
	})

	t.Run("taken", func(t *testing.T) {
		t.Parallel()

		u := New(&redisStub{hgetFn: func(_ context.Context, _, _ string) (string, error) {
			return "hash", nil
		}})

		isFree, err := u.IsUsernameFree(context.Background(), "alice")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if isFree {
			t.Fatal("expected username to be taken")
		}
	})
}

func TestGetUsers(t *testing.T) {
	t.Parallel()

	u := New(&redisStub{hgetAllFn: func(_ context.Context, key string) (map[string]string, error) {
		if key != userAuthKey {
			t.Fatalf("unexpected HGetAll key: %q", key)
		}

		return map[string]string{
			"bob":   "hash2",
			"alice": "hash1",
		}, nil
	}})

	got, err := u.GetUsers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Fatalf("expected sorted users [alice bob], got %v", got)
	}
}
