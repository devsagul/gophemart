package infra

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
)

func (app *App) registerUser(w http.ResponseWriter, r *http.Request) error {
	var data userRegisterRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if data.Login == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	user, err := core.NewUser(data.Login, data.Password)

	if err != nil {
		return err
	}

	err = app.store.WithContext(r.Context()).CreateUser(user)
	switch err.(type) {
	case *storage.ErrConflictingUserLogin:
		w.WriteHeader(http.StatusConflict)
	case nil:
	default:
		return err
	}

	err = app.login(user, w)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (app *App) loginUser(w http.ResponseWriter, r *http.Request) error {
	var data userLoginRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if data.Login == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	user, err := app.store.WithContext(r.Context()).ExtractUser(data.Login)

	switch err.(type) {
	case nil:
	case *storage.ErrUserNotFound:
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	default:
		return err
	}

	passwordIsValid, err := user.ValidatePassword(data.Password)
	if err != nil {
		return err
	}

	if !passwordIsValid {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}

	err = app.login(user, w)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (app *App) createOrder(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	orderID := string(body)

	order, err := core.NewOrder(orderID, user, time.Now())
	switch err.(type) {
	case *core.ErrInvalidOrder:
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	case nil:
	default:
		return err
	}

	err = app.store.WithContext(r.Context()).CreateOrder(order)
	switch err.(type) {
	case *storage.ErrOrderExists:
		w.WriteHeader(http.StatusOK)
		return nil
	case *storage.ErrOrderIDCollission:
		w.WriteHeader(http.StatusConflict)
		return nil
	case nil:
	default:
		return err
	}

	select {
	case app.accrualStream <- order:
	default:
	}

	w.WriteHeader(http.StatusAccepted)
	return nil
}

func (app *App) listOrders(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	orders, err := app.store.WithContext(r.Context()).ExtractOrdersByUser(user)

	if err != nil {
		return err
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	body, err := json.Marshal(orders)
	if err != nil {
		return err
	}
	wrapWrite(w, body)
	return nil
}

func (app *App) getBalance(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	witdrawn, err := app.store.WithContext(r.Context()).TotalWithdrawnSum(user)
	if err != nil {
		return err
	}

	type balanceResponse struct {
		Current   decimal.Decimal `json:"current"`
		Withdrawn decimal.Decimal `json:"withdrawn"`
	}

	data := balanceResponse{
		user.Balance,
		witdrawn,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	wrapWrite(w, body)
	return nil
}

func (app *App) createWithdrawal(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	var data WithdrawalRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	if data.Order == "" || data.Sum.LessThanOrEqual(decimal.Zero) {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	timestamp := time.Now()

	order, err := core.NewOrder(data.Order, user, timestamp)
	switch err.(type) {
	case *core.ErrInvalidOrder:
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	case nil:
	default:
		return err
	}
	withdrawal, err := core.NewWithdrawal(order, data.Sum, timestamp)
	if err != nil {
		return err
	}

	err = app.store.WithContext(r.Context()).CreateWithdrawal(withdrawal, order)

	switch err.(type) {
	case nil:
	case *storage.ErrOrderExists:
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	case *storage.ErrOrderIDCollission:
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	case *storage.ErrBalanceExceeded:
		w.WriteHeader(http.StatusPaymentRequired)
		return nil
	default:
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (app *App) listWithdrawals(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	withdrawals, err := app.store.WithContext(r.Context()).ExtractWithdrawalsByUser(user)
	if err != nil {
		return err
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	body, err := json.Marshal(withdrawals)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	wrapWrite(w, body)
	return nil
}
