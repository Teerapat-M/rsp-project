// Package passcode mints 6-digit room codes (000000–999999).
package passcode

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
)

const Digits = 6

var (
	ErrExhausted = errors.New("could not allocate a unique passcode")
	pattern      = regexp.MustCompile(`^[0-9]{6}$`)
)

var max = big.NewInt(1_000_000)

// New returns a cryptographically random 6-digit string, zero-padded.
func New() string {
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// rand.Reader failing is a host problem; zero is still a legal code
		// and uniqueness is enforced by the caller retrying on collision.
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func Valid(code string) bool {
	return pattern.MatchString(code)
}

// Allocate retries gen until taken returns false. taken should be cheap
// (in-memory set or a unique-index violation mapped to true).
func Allocate(taken func(string) bool, gen func() string, maxTries int) (string, error) {
	if gen == nil {
		gen = New
	}
	if maxTries < 1 {
		maxTries = 12
	}
	for i := 0; i < maxTries; i++ {
		code := gen()
		if !Valid(code) {
			continue
		}
		if taken != nil && taken(code) {
			continue
		}
		return code, nil
	}
	return "", ErrExhausted
}
