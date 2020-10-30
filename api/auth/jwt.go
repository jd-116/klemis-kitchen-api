package auth

import (
	"encoding/base64"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// JWTManager contains the secret loaded from the environment
type JWTManager struct {
	secret []byte
}

// Claims contains the data used to store a JWT's associated session info
type Claims struct {
	Username     string            `json:"u"`
	FirstName    string            `json:"fn"`
	LastName     string            `json:"ln"`
	IssuedAt     time.Time         `json:"i"`
	ExpiresAfter *int64            `json:"ea"`
	Permissions  types.Permissions `json:"p"`
}

// NewClaims combines a session and permission object
func NewClaims(session types.Session, permissions types.Permissions) *Claims {
	return &Claims{
		Username:     session.Username,
		FirstName:    session.FirstName,
		LastName:     session.LastName,
		IssuedAt:     session.IssuedAt,
		ExpiresAfter: session.ExpiresAfter,
		Permissions:  permissions,
	}
}

// Session extracts the type.Session value from the JWT claims
func (c *Claims) Session() *types.Session {
	return &types.Session{
		Username:     c.Username,
		FirstName:    c.FirstName,
		LastName:     c.LastName,
		IssuedAt:     c.IssuedAt,
		ExpiresAfter: c.ExpiresAfter,
	}
}

// Valid determines if the claims struct is valid by ensuring it has a username
// and that the issued at date + expires after is before today
func (c *Claims) Valid() error {
	if c.Username == "" {
		return errors.New("claims cannot have empty username")
	}

	// Make sure the claim has not expired
	if c.ExpiresAfter != nil {
		expiresAt := c.IssuedAt.Add(time.Duration(*c.ExpiresAfter) * time.Hour)
		if expiresAt.After(time.Now()) {
			return errors.New("claims are expired")
		}
	}

	return nil
}

// NewJWTManager creates a new JWTManager
// and loads the secret from the environment
func NewJWTManager() (*JWTManager, error) {
	jwtSecretStr, err := util.GetEnv("auth JWT secret key", "AUTH_JWT_SECRET")
	if err != nil {
		return nil, err
	}

	// Parse the string into bytes
	encoding := base64.StdEncoding.WithPadding(base64.StdPadding)
	secretBytes, err := encoding.DecodeString(jwtSecretStr)
	if err != nil {
		return nil, err
	}

	return &JWTManager{
		secret: secretBytes,
	}, nil
}

// IssueJWT creates and signs a new JWT for the given name/permissions
func (m *JWTManager) IssueJWT(session types.Session, permissions types.Permissions) *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, NewClaims(session, permissions))
}

// SignToken signs a JWT using the internal secret
func (m *JWTManager) SignToken(token *jwt.Token) (string, error) {
	// Sign and get the complete encoded token as a string
	// using the secret
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}

	return tokenString, err
}
