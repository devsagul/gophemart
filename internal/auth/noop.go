package auth

import (
	"net/http"

	"github.com/devsagul/gophemart/internal/core"
)

type NoopAuthProvider struct{}

func (noop NoopAuthProvider) Login(user *core.User) error {
	return nil
}

func (noop NoopAuthProvider) Logout(user *core.User) error {
	return nil
}

type NoopAuthBackend struct{}

func (noop NoopAuthBackend) GetAuthProvider(w http.ResponseWriter) AuthProvider {
	return NoopAuthProvider{}
}
