package users

import (
	"context"
	"fmt"
)

const userAuthKey = "user_auth"

type redis interface {
	HGet(ctx context.Context, key, field string) (string, error)
}

type Users struct {
	redis redis
}

func New(redis redis) *Users {
	return &Users{redis: redis}
}

func (u *Users) GetPasswordHash(ctx context.Context, userName string) (string, error) {
	pswd, err := u.redis.HGet(ctx, userAuthKey, userName)
	if err != nil {
		return "", fmt.Errorf("[users] failed to get password: %w", err)
	}

	return pswd, nil
}
