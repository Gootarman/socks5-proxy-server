package users

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Stat struct {
	Username   string
	UsageBytes int64
	Usage      string
	LastAuth   string
}

func (u *Users) Create(ctx context.Context, username, password string) error {
	if _, err := u.redis.HGet(ctx, userAuthKey, username); err == nil {
		return fmt.Errorf("[users] user with provided username already exists")
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
		return fmt.Errorf("[users] user with provided username not found")
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
			Usage:      formatBytes(raw.Usage),
			LastAuth:   lastLogin[raw.Username],
		})
	}

	return stats, nil
}

func formatBytes(size int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case size > gb:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(gb))
	case size > mb:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(mb))
	case size > kb:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
