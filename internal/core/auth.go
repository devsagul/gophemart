package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/devsagul/gophemart/internal/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const KeyLength = 64
const KeyPeriod = time.Duration(30 * 24 * time.Hour)
const KeyRefreshPeriod = time.Duration(6 * time.Hour)
const TokenPeriod = time.Duration(3 * time.Hour)

type ErrExpiredToken struct {
	expiredAt time.Time
}

func (err *ErrExpiredToken) Error() string {
	return fmt.Sprintf("token had expired at: %s", err.expiredAt)
}

type ErrUnexpectedSigningMethod struct {
	signingMethod jwt.SigningMethod
}

func (err *ErrUnexpectedSigningMethod) Error() string {
	return fmt.Sprintf("unexpected signing method: %s. HS256 expected", err.signingMethod.Alg())
}

type HmacKey struct {
	ID        uuid.UUID
	Sign      []byte
	ExpiresAt time.Time
}

func (key *HmacKey) Expired() bool {
	return key.ExpiresAt.Before(time.Now())
}

func (key *HmacKey) Fresh() bool {
	return time.Now().Before(key.ExpiresAt.Add(-4 * KeyRefreshPeriod))
}

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

type JwtClaims struct {
	UserID uuid.UUID `json:"user,omitempty"`
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
