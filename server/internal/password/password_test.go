package password

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Cheap parameters keep the suite fast; production parameters are covered by
// TestHashUsesDefaultParams.
var fastParams = Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=131072,t=4,p=4$") {
		t.Fatalf("Hash() produced an unexpected PHC string: %q", hash)
	}
	if err := Verify("correct horse battery staple", hash); err != nil {
		t.Fatalf("Verify() rejected the original password: %v", err)
	}
	if err := Verify("wrong password", hash); !errors.Is(err, ErrMismatch) {
		t.Fatalf("Verify() error = %v, want ErrMismatch", err)
	}
}

func TestHashUsesRandomSalt(t *testing.T) {
	a, _ := hashWith("same", fastParams)
	b, _ := hashWith("same", fastParams)
	if a == b {
		t.Fatal("two hashes of the same password are identical: salt is not random")
	}
}

func TestVerifyReadsParametersFromHash(t *testing.T) {
	// A hash created with other parameters (e.g. before a cost upgrade) must
	// still verify, because Verify reads m, t and p from the PHC string.
	custom := Params{Memory: 16 * 1024, Iterations: 2, Parallelism: 2, SaltLength: 24, KeyLength: 48}
	hash, err := hashWith("legacy password", custom)
	if err != nil {
		t.Fatal(err)
	}
	p, salt, key, err := Decode(hash)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if p != custom || len(salt) != 24 || len(key) != 48 {
		t.Fatalf("Decode() = %+v salt=%d key=%d, want %+v", p, len(salt), len(key), custom)
	}
	if err := Verify("legacy password", hash); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestDecodeRejectsMalformedHashes(t *testing.T) {
	valid, _ := hashWith("x", fastParams)
	parts := strings.Split(valid, "$")
	with := func(index int, value string) string {
		copied := append([]string{}, parts...)
		copied[index] = value
		return strings.Join(copied, "$")
	}
	tests := map[string]string{
		"empty":            "",
		"bcrypt":           "$2a$10$abcdefghijklmnopqrstuuMd1y0w6xD6bJb6j8l8p3oZ5x5x5x5x5",
		"argon2i variant":  with(1, "argon2i"),
		"wrong version":    with(2, "v=16"),
		"version garbage":  with(2, "v=19x"),
		"params garbage":   with(3, "m=8192,t=1,p=1x"),
		"missing parallel": with(3, "m=8192,t=1"),
		"zero memory":      with(3, "m=0,t=1,p=1"),
		"huge memory":      with(3, "m=99999999,t=1,p=1"),
		"huge iterations":  with(3, "m=8192,t=999,p=1"),
		"bad salt base64":  with(4, "!!!"),
		"short salt":       with(4, "YWJj"),
		"bad key base64":   with(5, "!!!"),
		"short key":        with(5, "YWJjZGVm"),
		"missing segment":  strings.Join(parts[:5], "$"),
		"extra segment":    valid + "$extra",
	}
	for name, hash := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := Decode(hash); !errors.Is(err, ErrInvalidHash) {
				t.Fatalf("Decode(%q) error = %v, want ErrInvalidHash", hash, err)
			}
			if err := Verify("x", hash); err == nil {
				t.Fatalf("Verify() accepted malformed hash %q", hash)
			}
		})
	}
}

func TestVerifyDummyTakesAsLongAsARealCheck(t *testing.T) {
	hash, _ := Hash("real password")
	VerifyDummy("warm up") // builds the dummy hash once

	start := time.Now()
	_ = Verify("wrong password", hash)
	real := time.Since(start)

	start = time.Now()
	VerifyDummy("wrong password")
	dummy := time.Since(start)

	// Both run the same Argon2id cost; allow generous scheduling noise.
	if dummy < real/3 {
		t.Fatalf("VerifyDummy() took %v, a real Verify took %v: unknown emails would be detectable", dummy, real)
	}
}

func TestValidateStrength(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"Str0ng!Passw0rd", true},
		{"Éléphant-2024!", true},
		{"Short1!a", false},
		{"alllowercase1!", false},
		{"ALLUPPERCASE1!", false},
		{"NoDigitsHere!!", false},
		{"NoSymbols12345", false},
		{strings.Repeat("Aa1!", 33), false},
	}
	for _, tt := range tests {
		err := ValidateStrength(tt.password)
		if (err == nil) != tt.valid {
			t.Errorf("ValidateStrength(%q) error = %v, want valid=%v", tt.password, err, tt.valid)
		}
	}
}
