package proxy

import (
	"context"
	"errors"
	"testing"
)

func TestAuth_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     string
		password string
		setup    func(t *testing.T, getter *PasswordHashGetterMock, comparator *PasswordComparatorMock)
		want     bool
	}{
		{
			name:     "hash getter error",
			user:     "alice",
			password: "secret",
			setup: func(t *testing.T, getter *PasswordHashGetterMock, _ *PasswordComparatorMock) {
				t.Helper()

				getter.GetPasswordHashMock.
					Expect(context.TODO(), "alice").
					Return("", errors.New("lookup failed"))
			},
			want: false,
		},
		{
			name:     "comparator error",
			user:     "bob",
			password: "guess",
			setup: func(t *testing.T, getter *PasswordHashGetterMock, comparator *PasswordComparatorMock) {
				t.Helper()

				getter.GetPasswordHashMock.
					Expect(context.TODO(), "bob").
					Return("hash", nil)
				comparator.ValidMock.
					Expect("guess", "hash").
					Return(false, errors.New("compare failed"))
			},
			want: false,
		},
		{
			name:     "invalid password",
			user:     "carol",
			password: "bad",
			setup: func(t *testing.T, getter *PasswordHashGetterMock, comparator *PasswordComparatorMock) {
				t.Helper()

				getter.GetPasswordHashMock.
					Expect(context.TODO(), "carol").
					Return("hash", nil)
				comparator.ValidMock.
					Expect("bad", "hash").
					Return(false, nil)
			},
			want: false,
		},
		{
			name:     "valid password",
			user:     "dave",
			password: "good",
			setup: func(t *testing.T, getter *PasswordHashGetterMock, comparator *PasswordComparatorMock) {
				t.Helper()

				getter.GetPasswordHashMock.
					Expect(context.TODO(), "dave").
					Return("hash", nil)
				comparator.ValidMock.
					Expect("good", "hash").
					Return(true, nil)
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getter := NewPasswordHashGetterMock(t)
			comparator := NewPasswordComparatorMock(t)
			tt.setup(t, getter, comparator)

			auth := NewAuth(getter, comparator)
			got := auth.Valid(tt.user, tt.password, "ignored")
			if got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthWithCache_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     string
		password string
		setup    func(t *testing.T, cache *CacheMock, validator *AuthValidatorMock)
		want     bool
	}{
		{
			name:     "cache hit",
			user:     "alice",
			password: "cached",
			setup: func(t *testing.T, cache *CacheMock, _ *AuthValidatorMock) {
				t.Helper()

				cache.GetMock.
					Expect(AuthCacheKey{user: "alice", password: "cached"}).
					Return(false, true)
			},
			want: false,
		},
		{
			name:     "cache miss",
			user:     "bob",
			password: "fresh",
			setup: func(t *testing.T, cache *CacheMock, validator *AuthValidatorMock) {
				t.Helper()

				key := AuthCacheKey{user: "bob", password: "fresh"}
				cache.GetMock.
					Expect(key).
					Return(false, false)
				validator.ValidMock.
					Expect("bob", "fresh", "").
					Return(true)
				cache.AddMock.
					Expect(key, true)
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache := NewCacheMock(t)
			validator := NewAuthValidatorMock(t)
			tt.setup(t, cache, validator)

			auth := NewAuthWithCache(cache, validator)
			got := auth.Valid(tt.user, tt.password, "ignored")
			if got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
