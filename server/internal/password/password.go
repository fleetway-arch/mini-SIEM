package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// Params are the Argon2id cost parameters. They are written into every hash
// in PHC format, so they can be raised later without breaking old hashes.
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultParams = Params{
	Memory:      128 * 1024,
	Iterations:  4,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// Upper bounds accepted when decoding a stored hash, so a corrupted or
// hostile hash cannot make Verify allocate unbounded memory or CPU.
const (
	maxMemory      = 512 * 1024
	maxIterations  = 16
	minKeyLength   = 16
	maxKeyLength   = 64
	minSaltLength  = 8
	maxSaltLength  = 64
	MinLength      = 12
	MaxLength      = 128
	maxConcurrency = 4
)

var (
	ErrMismatch    = errors.New("password does not match")
	ErrInvalidHash = errors.New("invalid argon2id hash")
)

// Each hash uses DefaultParams.Memory of RAM; limiting concurrent hashes
// keeps a burst of logins from exhausting memory.
var slots = make(chan struct{}, maxConcurrency)

func acquire() func() {
	slots <- struct{}{}
	return func() { <-slots }
}

func Hash(plainText string) (string, error) {
	return hashWith(plainText, DefaultParams)
}

func hashWith(plainText string, p Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	release := acquire()
	key := argon2.IDKey([]byte(plainText), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	release()
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify checks plainText against a PHC-formatted Argon2id hash, using the
// parameters stored in the hash and a constant-time comparison.
func Verify(plainText, encodedHash string) error {
	p, salt, expected, err := Decode(encodedHash)
	if err != nil {
		return err
	}
	release := acquire()
	actual := argon2.IDKey([]byte(plainText), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	release()
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return ErrMismatch
	}
	return nil
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// VerifyDummy runs a full Verify against a throwaway hash. Call it when the
// account does not exist so the response time matches a real check.
func VerifyDummy(plainText string) {
	dummyOnce.Do(func() {
		secret := make([]byte, 32)
		_, _ = rand.Read(secret)
		dummyHash, _ = Hash(base64.RawStdEncoding.EncodeToString(secret))
	})
	_ = Verify(plainText, dummyHash)
}

// Decode parses "$argon2id$v=19$m=65536,t=3,p=4$<salt>$<key>".
func Decode(encodedHash string) (Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version || parts[2] != fmt.Sprintf("v=%d", version) {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", p.Memory, p.Iterations, p.Parallelism) {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if p.Memory == 0 || p.Memory > maxMemory || p.Iterations == 0 || p.Iterations > maxIterations || p.Parallelism == 0 {
		return Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) < minSaltLength || len(salt) > maxSaltLength {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) < minKeyLength || len(key) > maxKeyLength {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(key))
	return p, salt, key, nil
}

// ValidateStrength applies the same rules as the signup form: at least 12
// characters with upper case, lower case, a digit and a symbol.
func ValidateStrength(plainText string) error {
	length := len([]rune(plainText))
	if length < MinLength || length > MaxLength {
		return fmt.Errorf("password must be between %d and %d characters", MinLength, MaxLength)
	}
	var upper, lower, digit, symbol bool
	for _, r := range plainText {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			symbol = true
		}
	}
	if !upper || !lower || !digit || !symbol {
		return errors.New("password must contain upper and lower case letters, a digit and a symbol")
	}
	return nil
}
