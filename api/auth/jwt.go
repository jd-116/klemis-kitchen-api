package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"

	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// JWTManager contains the secret loaded from the environment
type JWTManager struct {
	Auth       *jwtauth.JWTAuth
	secret     []byte
	BypassAuth bool
}

// Claims contains the data used to store a JWT's associated session info
type Claims struct {
	Username     string            `json:"sub"`
	FirstName    string            `json:"given_name"`
	LastName     string            `json:"family_name"`
	IssuedAt     time.Time         `json:"iat"`
	ExpiresAfter *int64            `json:"klemis:exa"`
	Permissions  types.Permissions `json:"klemis:perm"`
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
	jwtSecretStr, err := env.GetEnv("auth JWT secret key", "AUTH_JWT_SECRET")
	if err != nil {
		return nil, err
	}

	// Try to see if the server should bypass authentication
	bypassAuth := false
	if value, ok := os.LookupEnv("AUTH_BYPASS"); ok {
		if strings.TrimSpace(value) == "1" {
			bypassAuth = true
		}
	}

	// Parse the string into bytes
	encoding := base64.StdEncoding.WithPadding(base64.StdPadding)
	secretBytes, err := encoding.DecodeString(jwtSecretStr)
	if err != nil {
		return nil, err
	}

	// Create the instance of the auth used for middleware
	tokenAuth := jwtauth.New("HS256", secretBytes, nil)

	return &JWTManager{
		Auth:       tokenAuth,
		secret:     secretBytes,
		BypassAuth: bypassAuth,
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

type key int

// BypassAuthContextKey is the key to access the BypassAuth boolean field
// on request contexts that are processed by the Authenticated middleware
const BypassAuthContextKey key = iota

// Authenticated handles seeking, verifying, and validating JWT tokens,
// sending appropriate status codes upon failure.
func (m *JWTManager) Authenticated() func(http.Handler) http.Handler {
	// Seek, verify and validate JWT tokens
	verifier := jwtauth.Verify(m.Auth, jwtauth.TokenFromHeader)
	return func(next http.Handler) http.Handler {
		if m.BypassAuth {
			// Skip authentication
			verified := verifier(next)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Attach a value to the context
				ctx := context.WithValue(r.Context(), BypassAuthContextKey, true)

				// Pass it through
				verified.ServeHTTP(w, r.WithContext(ctx))
			})
		}

		// Compose the verifier and authenticator functions
		return verifier(authenticator(next))
	}
}

// AdminAuthenticated handles ensuring that the user has a valid token
// and is authorized (has sufficient permissions) to access admin resources
func AdminAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if value, ok := r.Context().Value(BypassAuthContextKey).(bool); ok && value == true {
			// Skip authentication
			next.ServeHTTP(w, r)
			return
		}

		_, claims, err := FromContext(r.Context())
		if err != nil {
			unauthorized(w)
			return
		}

		// Make sure the user has admin access
		if !claims.Permissions.AdminAccess {
			unauthorized(w)
			return
		}

		// User is authorized, pass it through
		next.ServeHTTP(w, r)
	})
}

// FromContext extracts the token and claims from the context
func FromContext(ctx context.Context) (*jwt.Token, *Claims, error) {
	token, _ := ctx.Value(jwtauth.TokenCtxKey).(*jwt.Token)
	err, _ := ctx.Value(jwtauth.ErrorCtxKey).(error)

	var claims *Claims = nil
	if token != nil {
		if tokenClaims, ok := token.Claims.(*Claims); ok {
			claims = tokenClaims
		} else {
			err = errors.New("invalid claim type")
		}
	}

	return token, claims, err
}

// authenticator sends an error response if token validation failed
func authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := FromContext(r.Context())

		if err != nil {
			unauthorized(w)
			return
		}

		if token == nil || !token.Valid {
			unauthorized(w)
			return
		}

		// Token is authenticated, pass it through
		next.ServeHTTP(w, r)
	})
}

// unauthorized sends a response message in the case that validation fails
func unauthorized(w http.ResponseWriter) {
	util.ErrorWithCode(w, errors.New("user is not authorized to access resource"),
		http.StatusUnauthorized)
}
