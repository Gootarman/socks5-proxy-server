package password

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValid(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		toCompare string
		wantOK    bool
		wantErr   bool
	}{
		{
			name:      "match",
			input:     "secret",
			toCompare: string(hash),
			wantOK:    true,
			wantErr:   false,
		},
		{
			name:      "mismatch",
			input:     "other",
			toCompare: string(hash),
			wantOK:    false,
			wantErr:   false,
		},
		{
			name:      "invalid hash",
			input:     "secret",
			toCompare: "not-a-bcrypt-hash",
			wantOK:    false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK, gotErr := New().Valid(tt.input, tt.toCompare)
			if gotOK != tt.wantOK {
				t.Fatalf("ok: got %v want %v", gotOK, tt.wantOK)
			}
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("err: got %v wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}
