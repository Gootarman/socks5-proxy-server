package createuser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	command     = "create-user"
	userAuthKey = "user_auth"
)

// TODO: переделать реализацию на работу с сервисным слоем, чтобы тут напрямую Redis не использовался
type redis interface {
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
}

type CommandHandler struct {
	redis redis
	in    *bufio.Reader
	out   io.Writer
}

func New(redis redis, in io.Reader, out io.Writer) *CommandHandler {
	if in == nil {
		in = os.Stdin
	}

	if out == nil {
		out = os.Stdout
	}

	return &CommandHandler{redis: redis, in: bufio.NewReader(in), out: out}
}

func (h *CommandHandler) CanHandle(_ context.Context, commandName string) bool {
	return commandName == command
}

//nolint:cyclop,gocyclo // CLI prompt flow is kept linear for readability.
func (h *CommandHandler) Handle(ctx context.Context) error {
	if h.redis == nil {
		return fmt.Errorf("[create-user] redis dependency is not configured")
	}

	if _, err := fmt.Fprint(h.out, "Input username and press Enter: "); err != nil {
		return fmt.Errorf("[create-user] failed to write username prompt: %w", err)
	}

	username, err := h.readInputLine()
	if err != nil {
		return fmt.Errorf("[create-user] failed to read username: %w", err)
	}

	if _, err = fmt.Fprint(h.out, "Input password and press Enter to create new user: "); err != nil {
		return fmt.Errorf("[create-user] failed to write password prompt: %w", err)
	}

	password, err := h.readInputLine()
	if err != nil {
		return fmt.Errorf("[create-user] failed to read password: %w", err)
	}

	if _, err = h.redis.HGet(ctx, userAuthKey, username); err == nil {
		return fmt.Errorf("[create-user] user with provided username already exists")
	} else if !errors.Is(err, goredis.Nil) {
		return fmt.Errorf("[create-user] failed to check if user exists: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("[create-user] failed to hash password: %w", err)
	}

	if err = h.redis.HSet(ctx, userAuthKey, username, string(hash)); err != nil {
		return fmt.Errorf("[create-user] failed to create user: %w", err)
	}

	if _, err = fmt.Fprintln(h.out, "User successfully created."); err != nil {
		return fmt.Errorf("[create-user] failed to write success message: %w", err)
	}

	return nil
}

func (h *CommandHandler) readInputLine() (string, error) {
	line, err := h.in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
