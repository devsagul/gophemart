package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	t.Run("create a user and check their password", func(t *testing.T) {
		user, err := NewUser("alice", "sikret")
		if err != nil {
			assert.FailNow(t, "could not create a user")
		}

		valid, err := user.ValidatePassword("sikret")
		if err != nil {
			assert.FailNow(t, "could not validate password")
		}
		assert.True(t, valid)

		valid, err = user.ValidatePassword("qwerty")
		if err != nil {
			assert.FailNow(t, "could not validate password")
		}
		assert.False(t, valid)
	})
	// set unusable password
}
