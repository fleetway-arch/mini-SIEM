package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func encodeSegment(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func validClaims(now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"uid": 7, "cid": 42, "role": "admin", "ver": 3, "iss": issuer,
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	}
}

func TestTokenManagerParse(t *testing.T) {
	manager := NewTokenManager(testSecret, 15*time.Minute)
	now := time.Now()

	valid, err := manager.Issue(7, 42, "admin", 3)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	expired, _ := NewTokenManager(testSecret, -time.Minute).Issue(7, 42, "admin", 3)
	otherSecret, _ := NewTokenManager("ffffffffffffffffffffffffffffffff", time.Minute).Issue(7, 42, "admin", 3)
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, validClaims(now)).SignedString([]byte(testSecret))
	none := encodeSegment(t, map[string]string{"alg": "none", "typ": "JWT"}) + "." + encodeSegment(t, validClaims(now)) + "."
	noneLib, _ := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims(now)).SignedString(jwt.UnsafeAllowNoneSignatureType)
	wrongIssuer := validClaims(now)
	wrongIssuer["iss"] = "someone-else"
	wrongIssuerToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongIssuer).SignedString([]byte(testSecret))
	noExpiry := validClaims(now)
	delete(noExpiry, "exp")
	noExpiryToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, noExpiry).SignedString([]byte(testSecret))
	missingClaims := validClaims(now)
	delete(missingClaims, "cid")
	missingClaimsToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, missingClaims).SignedString([]byte(testSecret))
	parts := strings.Split(valid, ".")
	tampered := parts[0] + "." + encodeSegment(t, jwt.MapClaims{"uid": 1, "cid": 42, "role": "owner", "ver": 3, "iss": issuer, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}) + "." + parts[2]

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid token", valid, false},
		{"expired", expired, true},
		{"wrong signature", otherSecret, true},
		{"tampered payload", tampered, true},
		{"wrong algorithm HS512", hs512, true},
		{"alg none handcrafted", none, true},
		{"alg none via library", noneLib, true},
		{"wrong issuer", wrongIssuerToken, true},
		{"missing expiry", noExpiryToken, true},
		{"missing company claim", missingClaimsToken, true},
		{"empty", "", true},
		{"garbage", "not.a.jwt", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.Parse(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() accepted an invalid token: %+v", claims)
				}
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if claims.UserID != 7 || claims.CompanyID != 42 || claims.Role != "admin" || claims.TokenVersion != 3 {
				t.Fatalf("Parse() claims = %+v", claims)
			}
			if claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.ExpiresAt.Sub(claims.IssuedAt.Time) != 15*time.Minute {
				t.Fatalf("Parse() iat/exp = %v/%v, want a 15m lifetime", claims.IssuedAt, claims.ExpiresAt)
			}
		})
	}
}
