package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrder(t *testing.T) {
	user, err := NewUser("Alice", "sikret")
	if err != nil {
		assert.FailNow(t, "Unable to instantiate user")
	}

	t.Run("Create a valid order", func(t *testing.T) {
		order, err := NewOrder("4561261212345467", user, time.Now())
		assert.NoError(t, err)
		if order == nil {
			assert.FailNow(t, "valid order creation returned nil as an order")
		}
		assert.Equal(t, NEW, order.Status)
	})

	t.Run("Create order with invalid id", func(t *testing.T) {
		testcases := []string{
			"",
			"order",
			"0rder",
			"42 order",
			"13 37",
			"4561261212345464",
		}
		msg := "order creation with id `%s` should return an error"
		for _, id := range testcases {
			_, err := NewOrder(id, user, time.Now())
			assert.Errorf(t, err, msg, id)
		}
	})

}

// test marshalling / unmarshalling logic
