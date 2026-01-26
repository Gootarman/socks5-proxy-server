package config

import "github.com/nskondratev/socks5-proxy-server/internal/env"

func RedisHost() string {
	return env.String("REDIS_HOST", "")
}

func RedisPort() int {
	return env.Int("REDIS_PORT", 6379)
}

func RedisDB() int {
	return env.Int("REDIS_DB", 0)
}
