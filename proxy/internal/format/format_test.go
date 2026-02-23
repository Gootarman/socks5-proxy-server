package format

import (
	"testing"
	"time"
)

func TestBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "bytes", size: 1024, want: "1024 B"},
		{name: "kb", size: 1025, want: "1.00 KB"},
		{name: "mb", size: 1048577, want: "1.00 MB"},
		{name: "gb", size: 1073741825, want: "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Bytes(tt.size)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFromNow(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		if got := FromNow(""); got != "-" {
			t.Fatalf("expected '-', got %q", got)
		}
	})

	t.Run("invalid date", func(t *testing.T) {
		t.Parallel()

		if got := FromNow("bad-date"); got != "bad-date" {
			t.Fatalf("expected original value, got %q", got)
		}
	})

	t.Run("valid date", func(t *testing.T) {
		t.Parallel()

		raw := time.Now().UTC().Add(-2 * time.Minute).Format(jsISOStringLayout)
		got := FromNow(raw)
		if got == "" || got == "-" || got == raw {
			t.Fatalf("expected human-readable relative value, got %q", got)
		}
	})
}
