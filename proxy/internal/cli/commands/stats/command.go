package stats

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
)

const (
	command = "users-stats"
)

type userService interface {
	GetStats(ctx context.Context) ([]users.Stat, error)
}

type CommandHandler struct {
	users userService
	out   io.Writer
}

func New(users userService, out io.Writer) *CommandHandler {
	if out == nil {
		out = os.Stdout
	}

	return &CommandHandler{users: users, out: out}
}

func (h *CommandHandler) CanHandle(_ context.Context, commandName string) bool {
	return commandName == command
}

func (h *CommandHandler) Handle(ctx context.Context) error {
	if h.users == nil {
		return fmt.Errorf("[users-stats] user service dependency is not configured")
	}

	stats, err := h.users.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("[users-stats] failed to get usage stats: %w", err)
	}

	header := "#\tUsername\tData usage (in bytes)\tData usage (human readable)\tLast login"

	tw := tabwriter.NewWriter(h.out, 0, 0, 2, ' ', 0)
	if _, err = fmt.Fprintln(tw, header); err != nil {
		return fmt.Errorf("[users-stats] failed to print header: %w", err)
	}

	for i, stat := range stats {
		loginDate := stat.LastAuth
		if loginDate == "" {
			loginDate = "-"
		}

		if _, err = fmt.Fprintf(
			tw,
			"%d\t%s\t%d\t%s\t%s\n",
			i+1,
			stat.Username,
			stat.UsageBytes,
			stat.Usage,
			loginDate,
		); err != nil {
			return fmt.Errorf("[users-stats] failed to print row: %w", err)
		}
	}

	if err = tw.Flush(); err != nil {
		return fmt.Errorf("[users-stats] failed to print stats: %w", err)
	}

	return nil
}
