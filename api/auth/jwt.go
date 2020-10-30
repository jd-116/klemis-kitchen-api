package auth

import (
	"encoding/base64"

	"github.com/dgrijalva/jwt-go"

	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// JWTManager contains the secret loaded from the environment
type JWTManager struct {
	secret []byte
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
func (m *JWTManager) IssueJWT(username string, permissions types.Permissions,
	firstName string, lastName string) (string, error) {

	// Construct the JWT with the claims map
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":  username,
		"fName": firstName,
		"lName": lastName,
		"perms": permissions,
	})

	// Sign and get the complete encoded token as a string
	// using the secret
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}

	return tokenString, err
}
