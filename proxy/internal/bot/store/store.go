package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

const (
	userStateKey = "user_state"
)

const (
	StateIdle                    = "idle"
	StateCreateUserEnterUsername = "create_user_enter_username"
	//nolint:gosec // False positive: state machine value, not a credential.
	StateCreateUserEnterPassword = "create_user_enter_password"
	StateDeleteUserEnterUsername = "delete_user_enter_username"
)

type redis interface {
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
}

type UserState struct {
	State string            `json:"state"`
	Data  map[string]string `json:"data"`
}

type UserStat struct {
	Num      int
	Username string
	UsageRaw int64
	Usage    string
	LastAuth string
}

type Store struct {
	redis redis
}

func New(redis redis) *Store {
	return &Store{
		redis: redis,
	}
}

func (s *Store) GetUserState(ctx context.Context, username string) (*UserState, error) {
	stateJSON, err := s.redis.HGet(ctx, userStateKey, username)
	if errors.Is(err, goredis.Nil) {
		//nolint:nilnil // Missing state is a valid case for callers.
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	var state UserState
	if err = json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}

	if state.Data == nil {
		state.Data = map[string]string{}
	}

	return &state, nil
}

func (s *Store) SetUserState(ctx context.Context, username string, state UserState) error {
	if state.Data == nil {
		state.Data = map[string]string{}
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal user state: %w", err)
	}

	if err = s.redis.HSet(ctx, userStateKey, username, stateJSON); err != nil {
		return fmt.Errorf("failed to save user state: %w", err)
	}

	return nil
}

func CleanPublicHost(raw string) string {
	withoutScheme := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")

	return strings.TrimSuffix(withoutScheme, "/")
}
