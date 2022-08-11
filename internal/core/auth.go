package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/devsagul/gophemart/internal/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const KEY_LENGTH = 64
const KEY_PERIOD = time.Duration(30 * 24 * time.Hour)
const KEY_REFRESH_PERIOD = time.Duration(6 * time.Hour)
const TOKEN_PERIOD = time.Duration(3 * time.Hour)

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
	Id        uuid.UUID
	sign      []byte
	expiresAt time.Time
}

func (key *HmacKey) Expired() bool {
	return key.expiresAt.Before(time.Now())
}

func (key *HmacKey) Fresh() bool {
	return time.Now().Before(key.expiresAt.Add(-4 * KEY_REFRESH_PERIOD))
}

func NewKey() (*HmacKey, error) {
	expiresAt := time.Now().Add(KEY_PERIOD)
	key := new(HmacKey)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	key.Id = id
	sign, err := utils.GenerateRandomBytes(KEY_LENGTH)
	if err != nil {
		return nil, err
	}
	key.sign = sign
	key.expiresAt = expiresAt
	return key, nil
}

type JwtClaims struct {
	UserId uuid.UUID `json:"user,omitempty"`
	jwt.StandardClaims
}

func (claims JwtClaims) Valid() error {
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return &ErrExpiredToken{time.Unix(claims.ExpiresAt, 0)}
	}
	return nil
}

func GenerateToken(user *User, key *HmacKey) (string, error) {
	now := time.Now()
	expiration := now.Add(time.Duration(TOKEN_PERIOD))

	claims := JwtClaims{
		user.Id,
		jwt.StandardClaims{
			ExpiresAt: expiration.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = key.Id.String()

	signed, err := token.SignedString(key.sign)

	if err != nil {
		return "", err
	}

	return signed, nil
}

func ParseToken(signed string, keys map[uuid.UUID]HmacKey) (userId uuid.UUID, err error) {
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

		keyId, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}

		key, found := keys[keyId]
		if !found {
			return nil, fmt.Errorf("key with id %s not found", keyId)
		}

		return key.sign, nil
	})

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims.UserId, err
	}

	return uuid.Nil, err
}
