package env

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func String(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultValue
}

func Int64(key string, defaultValue int64) int64 {
	rawVal := os.Getenv(key)

	parsedInt64, err := strconv.ParseInt(strings.TrimSpace(rawVal), 10, 64)
	if err != nil {
		return defaultValue
	}

	return parsedInt64
}

func Int(key string, defaultValue int) int {
	rawVal := os.Getenv(key)

	parsedInt, err := strconv.Atoi(strings.TrimSpace(rawVal))
	if err != nil {
		return defaultValue
	}

	return parsedInt
}

func Duration(key string, defaultValue time.Duration) time.Duration {
	parsedDuration, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return defaultValue
	}

	return parsedDuration
}

func StringURLEncoded(key, defaultValue string) string {
	decodedString, err := url.QueryUnescape(os.Getenv(key))
	if err != nil || decodedString == "" {
		return defaultValue
	}

	return decodedString
}

func Int64s(key string, defaultValue []int64) []int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}

	toParse := strings.Split(raw, ",")

	out := make([]int64, 0, len(toParse))

	for _, rawVal := range toParse {
		parsedInt64, err := strconv.ParseInt(strings.TrimSpace(rawVal), 10, 64)
		if err != nil {
			return defaultValue
		}

		out = append(out, parsedInt64)
	}

	return out
}

func Bool(key string, defaultValue bool) bool {
	parsed, err := strconv.ParseBool(String(key, ""))
	if err != nil {
		return defaultValue
	}

	return parsed
}
