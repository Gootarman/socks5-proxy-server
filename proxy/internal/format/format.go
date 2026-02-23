package format

import (
	"fmt"
	"time"
)

const jsISOStringLayout = "2006-01-02T15:04:05.000Z"

func Bytes(size int64) string {
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

func FromNow(raw string) string {
	if raw == "" {
		return "-"
	}

	t, err := time.Parse(jsISOStringLayout, raw)
	if err != nil {
		return raw
	}

	delta := time.Since(t)
	if delta < 0 {
		delta = -delta
	}

	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(delta.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(delta.Hours()/24))
	}
}
