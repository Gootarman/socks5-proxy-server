package env

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert.NoError(t, os.Setenv("TEST_STR_KEY", "test_val"))

	type args struct {
		key          string
		defaultValue string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Default value",
			args: args{
				key:          "NOTEXISTENT",
				defaultValue: "default_val",
			},
			want: "default_val",
		},
		{
			name: "Env value",
			args: args{
				key:          "TEST_STR_KEY",
				defaultValue: "test_val",
			},
			want: "test_val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := String(tt.args.key, tt.args.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration(t *testing.T) {
	type args struct {
		key          string
		defaultValue time.Duration
	}

	tests := []struct {
		name       string
		prepareEnv func(t *testing.T)
		args       args
		want       time.Duration
	}{
		{
			name: "Default value",
			args: args{
				key:          "NOTEXISTENT",
				defaultValue: time.Minute,
			},
			want: time.Minute,
		},
		{
			name: "Env value",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_DURATION_KEY", "1h"))
			},
			args: args{
				key:          "TEST_DURATION_KEY",
				defaultValue: time.Minute,
			},
			want: time.Hour,
		},
		{
			name: "Failed to parse duration, expect default value",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_DURATION_KEY", "some_bad"))
			},
			args: args{
				key:          "TEST_DURATION_KEY",
				defaultValue: time.Minute,
			},
			want: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareEnv != nil {
				tt.prepareEnv(t)
			}

			got := Duration(tt.args.key, tt.args.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringURLEncoded(t *testing.T) {
	type args struct {
		key          string
		defaultValue string
	}

	tests := []struct {
		name       string
		prepareEnv func(t *testing.T)
		args       args
		want       string
	}{
		{
			name: "Env value",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_STR_ENCODED_KEY", "%2Fsome%2Ftest%3Fparam%3Dval"))
			},
			args: args{
				key:          "TEST_STR_ENCODED_KEY",
				defaultValue: "",
			},
			want: "/some/test?param=val",
		},
		{
			name: "Default value",
			args: args{
				key:          "NOT_EXISTENT",
				defaultValue: "test_val",
			},
			want: "test_val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareEnv != nil {
				tt.prepareEnv(t)
			}

			got := StringURLEncoded(tt.args.key, tt.args.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt64s(t *testing.T) {
	type args struct {
		key          string
		defaultValue []int64
	}

	tests := []struct {
		name       string
		prepareEnv func(t *testing.T)
		args       args
		want       []int64
	}{
		{
			name: "Two ints from env",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_INT64S_KEY", "11, 22"))
			},
			args: args{
				key:          "TEST_INT64S_KEY",
				defaultValue: nil,
			},
			want: []int64{11, 22},
		},
		{
			name: "One int from env",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_INT64S_KEY", "11"))
			},
			args: args{
				key:          "TEST_INT64S_KEY",
				defaultValue: nil,
			},
			want: []int64{11},
		},
		{
			name: "Default value",
			args: args{
				key:          "NOT_EXISTENT",
				defaultValue: []int64{1},
			},
			want: []int64{1},
		},
		{
			name: "Failed to parse int, expect default value",
			prepareEnv: func(t *testing.T) {
				assert.NoError(t, os.Setenv("TEST_INT64S_KEY", "11,some_bad"))
			},
			args: args{
				key:          "TEST_INT64S_KEY",
				defaultValue: []int64{1},
			},
			want: []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareEnv != nil {
				tt.prepareEnv(t)
			}

			got := Int64s(tt.args.key, tt.args.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}
