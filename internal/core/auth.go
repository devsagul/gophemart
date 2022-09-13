// Auth contains different utils related to authentication and authorization

package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/devsagul/gophemart/internal/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// Length of the key (in bytes)
const KeyLength = 64

// Secret key lifespan
const KeyPeriod = time.Duration(30 * 24 * time.Hour)

// Timespan in which key has to be refreshed prior to its spoilage
const KeyRefreshPeriod = time.Duration(6 * time.Hour)

// Lifespan of a token
const TokenPeriod = time.Duration(3 * time.Hour)

// Error: token is expired
type ErrExpiredToken struct {
	expiredAt time.Time
}

// Formats ErrExpiredToken
func (err *ErrExpiredToken) Error() string {
	return fmt.Sprintf("token had expired at: %s", err.expiredAt)
}

// Error: signing method is unexpected (differs from HS256)
type ErrUnexpectedSigningMethod struct {
	signingMethod jwt.SigningMethod
}

// Formats ErrUnexpectedSigningMethod
func (err *ErrUnexpectedSigningMethod) Error() string {
	return fmt.Sprintf("unexpected signing method: %s. HS256 expected", err.signingMethod.Alg())
}

// Secret HMAC key
type HmacKey struct {
	// Key ID
	ID uuid.UUID

	// Key body (bytes)
	Sign []byte

	// Key's expiration datetime
	ExpiresAt time.Time
}

// Checks if secret key is expired
func (key *HmacKey) Expired() bool {
	return key.ExpiresAt.Before(time.Now())
}

// Check if secret key is fresh -> can be used to sign new tokens
func (key *HmacKey) Fresh() bool {
	return time.Now().Before(key.ExpiresAt.Add(-4 * KeyRefreshPeriod))
}

// Create new secret key
func NewKey() (*HmacKey, error) {
	expiresAt := time.Now().Add(KeyPeriod)
	key := new(HmacKey)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	key.ID = id
	sign, err := utils.GenerateRandomBytes(KeyLength)
	if err != nil {
		return nil, err
	}
	key.Sign = sign
	key.ExpiresAt = expiresAt
	return key, nil
}

// Custom JWT claims used in application
type JwtClaims struct {
	// User ID
	UserID uuid.UUID `json:"user,omitempty"`

	// default claims
	jwt.RegisteredClaims
}

func (claims JwtClaims) Valid() error {
	if !claims.VerifyExpiresAt(time.Now(), true) {
		return &ErrExpiredToken{claims.ExpiresAt.Time}
	}
	return nil
}

func GenerateToken(user *User, key *HmacKey) (string, error) {
	now := time.Now()
	expiration := now.Add(time.Duration(TokenPeriod))

	claims := JwtClaims{
		user.ID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = key.ID.String()

	signed, err := token.SignedString(key.Sign)

	if err != nil {
		return "", err
	}

	return signed, nil
}

func ParseToken(signed string, keys map[uuid.UUID]HmacKey) (userID uuid.UUID, err error) {
	token, err := jwt.ParseWithClaims(signed, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, &ErrUnexpectedSigningMethod{token.Method}
		}

		kid, found := token.Header["kid"]
		if !found {
			return nil, errors.New("no key id provided for token validation")
		}

		id, ok := kid.(string)
		if !ok {
			return nil, errors.New("key id should be provided as string")
		}

		keyID, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}

		key, found := keys[keyID]
		if !found {
			return nil, fmt.Errorf("key with id %s not found", keyID)
		}

		return key.Sign, nil
	})

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims.UserID, err
	}

	return uuid.Nil, err
}
