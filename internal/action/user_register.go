package action

import (
	"github.com/devsagul/gophemart/internal/auth"
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func UserRegister(username string, password string, store storage.UserStorage, authProvider auth.AuthProvider) error {
	user, err := core.NewUser(username, password)

	if err != nil {
		return err
	}

	// process uuid collission
	err = store.Create(user)
	if err != nil {
		return err
	}

	err = authProvider.Login(user)
	return err
}
