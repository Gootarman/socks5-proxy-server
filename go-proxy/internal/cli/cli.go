package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/nskondratev/socks5-proxy-server/internal/config"
	"github.com/nskondratev/socks5-proxy-server/internal/log"
)

type CLICommandsDeps struct{}

type commandHandler interface {
	CanHandle(ctx context.Context, commandName string) bool
	Handle(ctx context.Context) error
}

func HandleCLICommand(ctx context.Context, deps *CLICommandsDeps) (handled bool) {
	if len(os.Args) < 2 || os.Args[1] == "" {
		return handled
	}

	log.SetDefaultWithParams(log.OutputText, log.ParseStringLogLevel(config.LogLevel()))

	handled = true
	commandName := os.Args[1]

	fmt.Printf("In CLI command mode, process command with name: %s\n", commandName)

	commands := []commandHandler{}

	for i := range commands {
		if commands[i].CanHandle(ctx, commandName) {
			if err := commands[i].Handle(ctx); err != nil {
				fmt.Printf("Failed to handle command %s, error = %s\n", commandName, err.Error())

				return handled
			}
		}
	}

	fmt.Printf("Unknown command: %s\n", commandName)

	return handled
}
