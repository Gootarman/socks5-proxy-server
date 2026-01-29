//go:generate minimock -i .passwordHashGetter,.passwordComparator,.authValidator,.cache

package proxy

import (
	"context"
	"log"
)

type passwordHashGetter interface {
	GetPasswordHash(ctx context.Context, userName string) (string, error)
}

type passwordComparator interface {
	Valid(input, toCompare string) (bool, error)
}

type authValidator interface {
	Valid(user, password, userAddr string) bool
}

type Auth struct {
	passwordHashGetter passwordHashGetter
	passwordComparator passwordComparator
}

func NewAuth(
	passwordHashGetter passwordHashGetter,
	passwordComparator passwordComparator,
) *Auth {
	return &Auth{
		passwordHashGetter: passwordHashGetter,
		passwordComparator: passwordComparator,
	}
}

func (a *Auth) Valid(user, password, _ string) bool {
	ctx := context.TODO()

	passwordHash, err := a.passwordHashGetter.GetPasswordHash(ctx, user)
	if err != nil {
		log.Printf("failed to get password hash for user %s: %s", user, err.Error())

		return false
	}

	valid, err := a.passwordComparator.Valid(password, passwordHash)
	if err != nil {
		log.Printf("failed to compare password hash for user %s: %s", user, err.Error())

		return false
	}

	return valid
}

type AuthCacheKey struct {
	user     string
	password string
}

type cache interface {
	Add(key AuthCacheKey, value bool)
	Get(key AuthCacheKey) (value, exists bool)
}

type AuthWithCache struct {
	cache cache
	authValidator
}

func NewAuthWithCache(cache cache, authValidator authValidator) *AuthWithCache {
	return &AuthWithCache{
		cache:         cache,
		authValidator: authValidator,
	}
}

func (a *AuthWithCache) Valid(user, password, _ string) bool {
	entryCacheKey := AuthCacheKey{user, password}

	cachedValue, ok := a.cache.Get(entryCacheKey)
	if ok {
		return cachedValue
	}

	calculatedValue := a.authValidator.Valid(user, password, "")

	a.cache.Add(entryCacheKey, calculatedValue)

	return calculatedValue
}
