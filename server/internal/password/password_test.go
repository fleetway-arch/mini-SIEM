package password

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Fatalf("Hash() produced a non-Argon2id hash: %q", hash)
	}
	if err := Verify("correct horse battery staple", hash); err != nil {
		t.Fatalf("Verify() rejected the original password: %v", err)
	}
	if err := Verify("wrong password", hash); err == nil {
		t.Fatal("Verify() accepted an incorrect password")
	}
}
