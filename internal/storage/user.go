package storage

import (
	"fmt"

	"github.com/devsagul/gophemart/internal/core"
)

type UserStorage interface {
	Create(user *core.User) error
	Extract(login string) (*core.User, error)
	// TODO add extract by id
}

type ErrUserNotFound struct {
	login string
}

func (err *ErrUserNotFound) Error() string {
	return fmt.Sprintf("could not find user with login %s", err.login)
}

type ErrConflictingUserLogin struct {
	login string
}

func (err *ErrConflictingUserLogin) Error() string {
	return fmt.Sprintf("conflicting user login %s", err.login)
}
