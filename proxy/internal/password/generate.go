package password

import (
	"crypto/rand"
	"math/big"
)

const (
	lowers = "abcdefghijklmnopqrstuvwxyz"
	uppers = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits = "0123456789"
)

func Generate(length int) (string, error) {
	if length < 3 {
		length = 3
	}

	allChars := lowers + uppers + digits
	result := []byte{lowers[0], uppers[0], digits[0]}

	for len(result) < length {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(allChars))))
		if err != nil {
			return "", err
		}

		result = append(result, allChars[idx.Int64()])
	}

	for i := len(result) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}

		j := int(jBig.Int64())
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}
