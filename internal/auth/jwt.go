package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/golang-jwt/jwt/v4"
)

type JwtClaims struct {
	// TODO use user id instead of login
	Login string
	jwt.StandardClaims
}

var ErrExpiredToken = errors.New("expired token")
var ErrNoToken = errors.New("no token provided")
var ErrUnexpectedSigningMethod = errors.New("unexpected signing method")

func (claims JwtClaims) Valid() error {
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return ErrExpiredToken
	}
	return nil
}

type JwtAuthProvider struct {
	*JwtAuthBackend
	w http.ResponseWriter
	r *http.Request
}

func (provider *JwtAuthProvider) Login(user *core.User) error {
	// TODO obtain key from database
	hmacKey := "sikret key"
	now := time.Now()
	expiration := now.Add(time.Duration(3 * time.Hour))

	claims := JwtClaims{
		user.Login,
		jwt.StandardClaims{
			ExpiresAt: expiration.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(hmacKey))

	if err != nil {
		return err
	}

	provider.w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", signed))

	return nil
}

func (provider *JwtAuthProvider) Auth() (*core.User, error) {
	header := provider.r.Header.Get("Authorization")
	var signed string
	_, err := fmt.Sscanf(header, "Bearer %s", &signed)
	if err != nil {
		return nil, ErrNoToken
	}

	token, err := jwt.ParseWithClaims(signed, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}

		return []byte("sikret key"), nil
	})

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		login := claims.Login
		user, err := provider.users.Extract(login)
		return user, err
	}

	return nil, err
}

type JwtAuthBackend struct {
	users storage.UserStorage
}

func NewJwtAutAuthBackend(users storage.UserStorage) *JwtAuthBackend {
	return &JwtAuthBackend{
		users,
	}
}

func (backend *JwtAuthBackend) GetAuthProvider(
	w http.ResponseWriter,
	r *http.Request,
) AuthProvider {
	return &JwtAuthProvider{
		backend,
		w,
		r,
	}
}
