package password

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

var params = &argon2id.Params{
	Memory:      128 * 1024,
	Iterations:  4,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

func Hash(plainText string) (string, error) {
	hash, err := argon2id.CreateHash(plainText, params)
	if err != nil {
		return "", fmt.Errorf("hash password with argon2id: %w", err)
	}
	return hash, nil
}

func Verify(plainText, encodedHash string) error {
	match, err := argon2id.ComparePasswordAndHash(plainText, encodedHash)
	if err != nil {
		return fmt.Errorf("verify password with argon2id: %w", err)
	}
	if !match {
		return fmt.Errorf("password does not match")
	}
	return nil
}
