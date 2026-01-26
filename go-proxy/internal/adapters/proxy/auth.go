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
