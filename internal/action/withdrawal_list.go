package action

import (
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
)

func WithdrawalList(user *core.User, store storage.Storage) ([]*core.Withdrawal, error) {
	return store.ExtractWithdrawalsByUser(user)
}
