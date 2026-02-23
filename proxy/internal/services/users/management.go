package users

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	formatter "github.com/nskondratev/socks5-proxy-server/proxy/internal/format"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists   = errors.New("user with provided username already exists")
	ErrUserNotFound = errors.New("user with provided username not found")
)

type Stat struct {
	Username   string
	UsageBytes int64
	Usage      string
	LastAuth   string
}

func (u *Users) Create(ctx context.Context, username, password string) error {
	if _, err := u.redis.HGet(ctx, userAuthKey, username); err == nil {
		return ErrUserExists
	} else if !errors.Is(err, goredis.Nil) {
		return fmt.Errorf("[users] failed to check if user exists: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("[users] failed to hash password: %w", err)
	}

	if err = u.redis.HSet(ctx, userAuthKey, username, string(hash)); err != nil {
		return fmt.Errorf("[users] failed to create user: %w", err)
	}

	return nil
}

func (u *Users) Delete(ctx context.Context, username string) error {
	if _, err := u.redis.HGet(ctx, userAuthKey, username); errors.Is(err, goredis.Nil) {
		return ErrUserNotFound
	} else if err != nil {
		return fmt.Errorf("[users] failed to check if user exists: %w", err)
	}

	if err := u.redis.HDel(ctx, userAuthKey, username); err != nil {
		return fmt.Errorf("[users] failed to delete user: %w", err)
	}

	return nil
}

func (u *Users) GetStats(ctx context.Context) ([]Stat, error) {
	dataUsage, err := u.redis.HGetAll(ctx, userUsageDataKey)
	if err != nil {
		return nil, fmt.Errorf("[users] failed to get usage stats: %w", err)
	}

	lastLogin, err := u.redis.HGetAll(ctx, userAuthDateKey)
	if err != nil {
		return nil, fmt.Errorf("[users] failed to get last login stats: %w", err)
	}

	type rawStat struct {
		Username string
		Usage    int64
	}

	rawStats := make([]rawStat, 0, len(dataUsage))
	for username, usageStr := range dataUsage {
		usage, parseErr := strconv.ParseInt(usageStr, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("[users] failed to parse usage for user %s: %w", username, parseErr)
		}

		rawStats = append(rawStats, rawStat{Username: username, Usage: usage})
	}

	sort.Slice(rawStats, func(i, j int) bool {
		return rawStats[i].Usage > rawStats[j].Usage
	})

	stats := make([]Stat, 0, len(rawStats))
	for _, raw := range rawStats {
		stats = append(stats, Stat{
			Username:   raw.Username,
			UsageBytes: raw.Usage,
			Usage:      formatter.Bytes(raw.Usage),
			LastAuth:   lastLogin[raw.Username],
		})
	}

	return stats, nil
}

func (u *Users) IsUsernameFree(ctx context.Context, username string) (bool, error) {
	_, err := u.redis.HGet(ctx, userAuthKey, username)
	if errors.Is(err, goredis.Nil) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("[users] failed to check if user exists: %w", err)
	}

	return false, nil
}

func (u *Users) GetUsers(ctx context.Context) ([]string, error) {
	usersMap, err := u.redis.HGetAll(ctx, userAuthKey)
	if err != nil {
		return nil, fmt.Errorf("[users] failed to get users: %w", err)
	}

	users := make([]string, 0, len(usersMap))
	for user := range usersMap {
		users = append(users, user)
	}

	sort.Strings(users)

	return users, nil
}
