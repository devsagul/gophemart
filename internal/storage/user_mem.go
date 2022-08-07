package storage

import (
	"sync"

	"github.com/devsagul/gophemart/internal/core"
)

type userMemStorage struct {
	sync.RWMutex
	m map[string]core.User
}

func (store *userMemStorage) Create(user *core.User) error {
	login := user.Login

	store.Lock()
	defer store.Unlock()

	_, found := store.m[login]
	if found {
		return &ErrConflictingUserLogin{login}
	}
	store.m[login] = *user
	return nil
}

func (store *userMemStorage) Extract(login string) (*core.User, error) {
	store.RLock()
	defer store.RUnlock()

	user, found := store.m[login]
	if !found {
		return nil, &ErrUserNotFound{login}
	}

	return &user, nil
}

func NewUserMemStorage() *userMemStorage {
	storage := new(userMemStorage)
	storage.m = make(map[string]core.User)
	return storage
}
