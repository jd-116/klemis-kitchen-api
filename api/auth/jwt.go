package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"

	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// JWTManager contains the secret loaded from the environment
type JWTManager struct {
	signer jwt.SigningMethod
	parser *jwt.Parser
	secret []byte
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

	// Parse the string into bytes
	encoding := base64.StdEncoding.WithPadding(base64.StdPadding)
	secretBytes, err := encoding.DecodeString(jwtSecretStr)
	if err != nil {
		return nil, err
	}

	return &JWTManager{
		signer: jwt.GetSigningMethod("H256"),
		parser: &jwt.Parser{},
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

type key int

// BypassAuthContextKey is the key to access the BypassAuth boolean field
// on request contexts that are processed by the Authenticated middleware
const BypassAuthContextKey key = iota

// Authenticated handles seeking, verifying, and validating JWT tokens,
// sending appropriate status codes upon failure.
func (m *JWTManager) Authenticated() func(http.Handler) http.Handler {
	// Seek, verify and validate JWT tokens
	verifier := m.verifier()
	return func(next http.Handler) http.Handler {
		// Compose the verifier and authenticator functions
		return verifier(authenticator(next))
	}
}

// AdminAuthenticated handles ensuring that the user has a valid token
// and is authorized (has sufficient permissions) to access admin resources
func AdminAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, err := FromContext(r.Context())
		if err != nil {
			log.Printf("error when getting claims from context: %s\n", err)
			unauthorized(w)
			return
		}

		// Make sure the user has admin access
		if !claims.Permissions.AdminAccess {
			log.Println("user lacks admin access")
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
			log.Printf("error when getting token from context: %s\n", err)
			unauthorized(w)
			return
		}

		if token == nil || !token.Valid {
			log.Println("token is nil or invalid")
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

// verifier is an HTTP middleware that verifies the JWT token in a request
func (m *JWTManager) verifier() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token, err := m.verifyRequest(r, jwtauth.TokenFromCookie)
			ctx = jwtauth.NewContext(ctx, token, err)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// verifyRequest attempts to parse the JWT token ina  request
func (m *JWTManager) verifyRequest(r *http.Request,
	findTokenFns ...func(r *http.Request) string) (*jwt.Token, error) {

	var tokenStr string
	var err error

	// Extract token string from the request by calling token find functions in
	// the order they where provided. Further extraction stops if a function
	// returns a non-empty string.
	for _, fn := range findTokenFns {
		tokenStr = fn(r)
		if tokenStr != "" {
			break
		}
	}
	if tokenStr == "" {
		return nil, jwtauth.ErrNoTokenFound
	}

	// Verify the token
	keyFunc := func(t *jwt.Token) (interface{}, error) { return m.secret, nil }
	token, err := m.parser.ParseWithClaims(tokenStr, &Claims{}, keyFunc)
	if err != nil {
		if verr, ok := err.(*jwt.ValidationError); ok {
			if verr.Errors&jwt.ValidationErrorExpired > 0 {
				return token, jwtauth.ErrExpired
			} else if verr.Errors&jwt.ValidationErrorIssuedAt > 0 {
				return token, jwtauth.ErrIATInvalid
			} else if verr.Errors&jwt.ValidationErrorNotValidYet > 0 {
				return token, jwtauth.ErrNBFInvalid
			}
		}
		return token, err
	}

	if token == nil || !token.Valid {
		err = jwtauth.ErrUnauthorized
		return token, err
	}

	// Verify signing algorithm
	if token.Method != m.signer {
		return token, jwtauth.ErrAlgoInvalid
	}

	// Valid!
	return token, nil
}
