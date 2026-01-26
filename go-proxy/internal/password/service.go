package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
}

func New() *Password {
	return &Password{}
}

func (p *Password) Valid(input, toCompare string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(toCompare), []byte(input))

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, fmt.Errorf("[password] failed to compare hash and password: %w", err)
	}
}
