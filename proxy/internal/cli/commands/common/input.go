package common

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

func ReadInputLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
