package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "mini-siem"

var ErrInvalidToken = errors.New("invalid or expired token")

// Claims carried by an access token. TokenVersion must match the user's
// current tokenVersion, which lets the server invalidate tokens early.
type Claims struct {
	UserID       int    `json:"uid"`
	CompanyID    int    `json:"cid"`
	Role         string `json:"role"`
	TokenVersion int    `json:"ver"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (m *TokenManager) TTL() time.Duration {
	return m.ttl
}

// Issue signs an HS256 access token for the given user.
func (m *TokenManager) Issue(userID, companyID int, role string, tokenVersion int) (string, error) {
	now := m.now()
	claims := Claims{
		UserID:       userID,
		CompanyID:    companyID,
		Role:         role,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.Itoa(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// Parse validates the signature, the algorithm (HS256 only, so "none" and
// any asymmetric algorithm are rejected), expiry and issuer.
func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(issuer),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.UserID == 0 || claims.CompanyID == 0 || claims.Role == "" {
		return nil, fmt.Errorf("%w: missing required claims", ErrInvalidToken)
	}
	return claims, nil
}
