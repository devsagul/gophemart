package infra

import "github.com/shopspring/decimal"

type userRegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userLoginRequest userRegisterRequest

type WithdrawalRequest struct {
	Order string          `json:"order"`
	Sum   decimal.Decimal `json:"sum"`
}
