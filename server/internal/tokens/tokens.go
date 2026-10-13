// Package tokens creates opaque random tokens (refresh, invitation, password
// reset). Only the SHA-256 hash is ever stored; the plain value goes to the
// client once.
package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const byteLength = 32

// New returns a URL-safe token built from 32 random bytes and its hash.
func New() (plain string, hash string, err error) {
	raw := make([]byte, byteLength)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	return plain, Hash(plain), nil
}

// Hash returns the hex SHA-256 digest used as the database lookup key.
func Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
