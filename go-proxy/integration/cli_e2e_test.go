//go:build integration

package integration

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/nskondratev/socks5-proxy-server/internal/cli"
)

func TestCLIE2E_AllCommands(t *testing.T) {
	redis := newFakeRedis()

	handled, out := runCLICommand(t, redis, "create-admin", "admin\n")
	if !handled {
		t.Fatal("expected create-admin to be handled")
	}
	if !strings.Contains(out, "Admin successfully created.") {
		t.Fatalf("unexpected create-admin output: %q", out)
	}

	isAdmin, err := redis.HExists(context.Background(), redisUserAdminKey, "admin")
	if err != nil {
		t.Fatalf("failed to check admin key: %v", err)
	}
	if !isAdmin {
		t.Fatal("admin user was not saved")
	}

	handled, out = runCLICommand(t, redis, "create-user", "alice\nsecret\n")
	if !handled {
		t.Fatal("expected create-user to be handled")
	}
	if !strings.Contains(out, "User successfully created.") {
		t.Fatalf("unexpected create-user output: %q", out)
	}

	hash, err := redis.HGet(context.Background(), redisUserAuthKey, "alice")
	if err != nil {
		t.Fatalf("failed to get user hash: %v", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}

	if err = redis.HSet(context.Background(), redisUserUsageKey, "alice", 2048); err != nil {
		t.Fatalf("failed to seed user stats: %v", err)
	}
	if err = redis.HSet(context.Background(), redisUserAuthDate, "alice", "2026-02-01T00:00:00.000Z"); err != nil {
		t.Fatalf("failed to seed auth date: %v", err)
	}

	handled, out = runCLICommand(t, redis, "users-stats", "")
	if !handled {
		t.Fatal("expected users-stats to be handled")
	}
	if !strings.Contains(out, "alice") || !strings.Contains(out, "2.00 KB") {
		t.Fatalf("unexpected users-stats output: %q", out)
	}

	handled, out = runCLICommand(t, redis, "delete-user", "alice\n")
	if !handled {
		t.Fatal("expected delete-user to be handled")
	}
	if !strings.Contains(out, "User successfully deleted.") {
		t.Fatalf("unexpected delete-user output: %q", out)
	}

	if _, err = redis.HGet(context.Background(), redisUserAuthKey, "alice"); err == nil {
		t.Fatal("user was not deleted")
	}

	handled, out = runCLICommand(t, redis, "delete-admin", "admin\n")
	if !handled {
		t.Fatal("expected delete-admin to be handled")
	}
	if !strings.Contains(out, "Admin successfully deleted.") {
		t.Fatalf("unexpected delete-admin output: %q", out)
	}

	isAdmin, err = redis.HExists(context.Background(), redisUserAdminKey, "admin")
	if err != nil {
		t.Fatalf("failed to check admin key after delete: %v", err)
	}
	if isAdmin {
		t.Fatal("admin user was not deleted")
	}
}

func runCLICommand(t *testing.T, redis *fakeRedis, command, input string) (bool, string) {
	t.Helper()

	oldArgs := os.Args
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	stdinFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("failed to create temp stdin file: %v", err)
	}
	if _, err = stdinFile.WriteString(input); err != nil {
		t.Fatalf("failed to write input data: %v", err)
	}
	if _, err = stdinFile.Seek(0, 0); err != nil {
		t.Fatalf("failed to reset input file cursor: %v", err)
	}
	defer func() {
		_ = stdinFile.Close()
	}()

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	defer func() {
		_ = readPipe.Close()
	}()
	defer func() {
		_ = writePipe.Close()
	}()

	outputCh := make(chan string, 1)
	go func() {
		defer close(outputCh)
		raw, _ := io.ReadAll(readPipe)
		outputCh <- string(raw)
	}()

	os.Args = []string{"go-proxy", command}
	os.Stdin = stdinFile
	os.Stdout = writePipe

	handled := cli.HandleCLICommand(context.Background(), &cli.CommandsDeps{Redis: redis})

	_ = writePipe.Close()
	output := <-outputCh

	return handled, output
}
