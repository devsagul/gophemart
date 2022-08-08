package auth

import (
	"net/http"

	"github.com/devsagul/gophemart/internal/core"
)

type AuthProvider interface {
	Login(*core.User) error
	Auth() (*core.User, error)
}

type AuthBackend interface {
	GetAuthProvider(http.ResponseWriter, *http.Request) AuthProvider
}
