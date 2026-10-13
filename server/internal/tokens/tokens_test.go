package tokens

import (
	"encoding/base64"
	"testing"
)

func TestNewReturnsRandom32ByteTokenAndItsHash(t *testing.T) {
	plain, hash, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(plain)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token decodes to %d bytes (err %v), want 32", len(raw), err)
	}
	if hash != Hash(plain) || len(hash) != 64 {
		t.Fatalf("hash = %q, want hex SHA-256 of the token", hash)
	}
	if hash == plain {
		t.Fatal("hash must differ from the plain token")
	}
	other, _, _ := New()
	if other == plain {
		t.Fatal("two tokens are identical")
	}
}
