package auth

import (
	"net/http"

	"github.com/devsagul/gophemart/internal/core"
)

type AuthProvider interface {
	Login(user *core.User) error
	Logout(user *core.User) error
}

type AuthBackend interface {
	GetAuthProvider(w http.ResponseWriter) AuthProvider
}
