package storage

import (
	"testing"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestUserStorageMem(t *testing.T) {
	user, err := core.NewUser("alice", "sikret")
	if err != nil {
		assert.FailNow(t, "could not create user")
	}
	store := NewUserMemStorage()
	err = store.Create(user)
	if err != nil {
		assert.FailNow(t, "could not persist user")
	}
	extracted, err := store.Extract(user.Login)
	if err != nil {
		assert.FailNow(t, "could not extract user")
	}
	if extracted == nil {
		assert.FailNow(t, "extracted user is nil")
	}
	assert.Equal(t, *user, *extracted)

}
