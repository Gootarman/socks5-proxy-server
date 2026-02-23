package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrEmptyInput = errors.New("input can not be empty")

func ReadInputLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func PromptAndReadRequiredInput(out io.Writer, in *bufio.Reader, prompt, fieldName string) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt: %w", err)
	}

	line, err := ReadInputLine(in)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	if line == "" {
		return "", fmt.Errorf("%w: %s", ErrEmptyInput, fieldName)
	}

	return line, nil
}

func WriteSuccess(out io.Writer, message string) error {
	if _, err := fmt.Fprintln(out, message); err != nil {
		return fmt.Errorf("failed to write success message: %w", err)
	}

	return nil
}
