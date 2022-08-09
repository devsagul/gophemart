package infra

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/devsagul/gophemart/internal/action"
	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
)

func (app *App) registerUser(w http.ResponseWriter, r *http.Request) {
	var data userRegisterRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if data.Login == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = action.UserRegister(data.Login, data.Password, app.store, app.auth.GetAuthProvider(w, r))
	switch err.(type) {
	case *storage.ErrConflictingUserLogin:
		w.WriteHeader(http.StatusConflict)
	case nil:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *App) loginUser(w http.ResponseWriter, r *http.Request) {
	var data userLoginRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if data.Login == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = action.UserLogin(data.Login, data.Password, app.store, app.auth.GetAuthProvider(w, r))
	switch err.(type) {
	case nil:
		w.WriteHeader(http.StatusOK)
	case *storage.ErrUserNotFound:
		w.WriteHeader(http.StatusUnauthorized)
	case *action.ErrInvalidPassword:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *App) createOrder(w http.ResponseWriter, r *http.Request) {
	user, err := app.auth.GetAuthProvider(w, r).Auth()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orderId := string(body)
	err = action.OrderCreate(orderId, user, app.store)
	if err == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if errors.Is(err, storage.ErrOrderExitst) {
		w.WriteHeader(http.StatusOK)
		return
	}
	if errors.Is(err, storage.ErrOrderIdCollision) {
		w.WriteHeader(http.StatusConflict)
		return
	}
	if errors.Is(err, core.ERR_INVALID_ORDER) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func (app *App) listOrders(w http.ResponseWriter, r *http.Request) {
	user, err := app.auth.GetAuthProvider(w, r).Auth()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := action.OrderList(user, app.store)
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(orders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		// log the error
	}
}

func (app *App) getBalance(w http.ResponseWriter, r *http.Request) {
	user, err := app.auth.GetAuthProvider(w, r).Auth()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	type balanceResponse struct {
		Current   decimal.Decimal `json:"current"`
		Withdrawn int             `json:"withdrawn"`
	}

	withdrawals, err := action.WithdrawalList(user, app.store)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := balanceResponse{
		user.Balance,
		len(withdrawals),
	}

	body, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		// log the error
	}
}

func (app *App) createWithdrawal(w http.ResponseWriter, r *http.Request) {
	user, err := app.auth.GetAuthProvider(w, r).Auth()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var data WithdrawalRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = action.WithdrawalCreate(user, data.Order, data.Sum, app.store)
	switch err.(type) {
	case nil:
		w.WriteHeader(http.StatusOK)
		return
	case *storage.ErrBalanceExceeded:
		w.WriteHeader(http.StatusPaymentRequired)
		return
	default:
		if err == core.ERR_INVALID_ORDER {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (app *App) listWithdrawals(w http.ResponseWriter, r *http.Request) {
	user, err := app.auth.GetAuthProvider(w, r).Auth()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := action.WithdrawalList(user, app.store)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := json.Marshal(withdrawals)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		// log error
	}
}
