package action

import (
	"github.com/devsagul/gophemart/internal/auth"
	"github.com/devsagul/gophemart/internal/storage"
)

type ErrInvalidPassword struct{}

func (*ErrInvalidPassword) Error() string {
	return "invalid password"
}

func UserLogin(username string, password string, store storage.UserStorage, authProvider auth.AuthProvider) error {
	user, err := store.Extract(username)

	if err != nil {
		return err
	}

	passwordIsValid, err := user.ValidatePassword(password)
	if err != nil {
		return err
	}

	if !passwordIsValid {
		return &ErrInvalidPassword{}
	}

	err = authProvider.Login(user)
	return err
}
