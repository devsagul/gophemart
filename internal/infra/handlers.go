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

// @Title Gophemart API
// @Description Service for registering and processing orders.
// @Version 1.0

// @Contact.email dev.sagul@gmail.com

// @BasePath /api
// @Host devsagul.github.io:8080

// @SecurityDefinitions.apikey ApiKeyAuth
// @In header
// @Name authorization

// @Tag.name Auth
// @Tag.description "Authentication related stuff"

// @Tag.name User
// @Tag.description "Group for general user-related endpoints"

// @Tag.name Orders
// @Tag.description "Group for orders-related endpoints"

// @Tag.name Withdrawals
// @Tag.description "Group for withdrawals-related endpoints"

// RegisterUser godoc
// @Tags Auth
// @Summary Request a user registration
// @Description Request a user registration with particular login and password
// @ID authRegisterUser
// @Accept  json
// @Produce json
// @Param login body string true "Login"
// @Param password body string true "Password"
// @Success 200 {empty}
// @Failure 409 {string} string "Conflicting user logins"
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Internal server error"
// @Router /user/register [post]
func (app *App) RegisterUser(w http.ResponseWriter, r *http.Request) error {
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

// LoginUser godoc
// @Tags Auth
// @Summary Request a user login
// @Description Request a user login with particular login and password
// @ID authLoginUser
// @Accept  json
// @Produce json
// @Param login body string true "Login"
// @Param password body string true "Password"
// @Success 200 {empty}
// @Failure 401 {string} string "Password or login is incorrect"
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Internal server error"
// @Router /user/login [post]
func (app *App) LoginUser(w http.ResponseWriter, r *http.Request) error {
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

// CreateOrder godoc
// @Tags Orders
// @Summary Request order creation
// @Description Request an order creation with particular id
// @ID orderCreateOrder
// @Accept  text
// @Produce json
// @Param id body string true "Order id"
// @Success 202 {empty}
// @Failure 401 {string} string "Not authenticated"
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Internal server error"
// @Security ApiKeyAuth
// @Router /user/orders [post]
func (app *App) CreateOrder(w http.ResponseWriter, r *http.Request) error {
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

// ListOrder godoc
// @Tags Orders
// @Summary Get all the orders
// @Description Get all the orders for current user
// @ID orderListOrder
// @Accept  json
// @Produce json
// @Success 200 {list} core.Order
// @Failure 204 {empty}
// @Failure 401 {string} string "Not authenticated"
// @Failure 500 {string} string "Internal server error"
// @Security ApiKeyAuth
// @Router /user/orders [get]
func (app *App) ListOrders(w http.ResponseWriter, r *http.Request) error {
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

// Get balance response
type BalanceResponse struct {
	// Current balance
	Current decimal.Decimal `json:"current"`
	// Total withdrawn amount
	Withdrawn decimal.Decimal `json:"withdrawn"`
}

// GetBalance godoc
// @Tags User
// @Summary Get the balance
// @Description Get the balance of the current user
// @ID userGetBalance
// @Accept  json
// @Produce json
// @Success 200 {object} BalanceResponse
// @Failure 401 {string} string "Not authenticated"
// @Failure 500 {string} string "Internal server error"
// @Security ApiKeyAuth
// @Router /user/balance [get]
func (app *App) GetBalance(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	witdrawn, err := app.store.WithContext(r.Context()).TotalWithdrawnSum(user)
	if err != nil {
		return err
	}

	data := BalanceResponse{
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

// CreateWithdrawal godoc
// @Tags Withdrawals
// @Summary Request withdrawal creation
// @Description Request withdrawal creation for concrete order of given user
// @ID userCreateWithdrawal
// @Accept  json
// @Produce json
// @Param order body string true "Order id"
// @Param sum body number true "Withdrawal sum"
// @Success 200 {empty}
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Not authenticated"
// @Failure 402 {string} string "User's balance exceeded"
// @Failure 422 {string} string "Unprocessable entity"
// @Failure 500 {string} string "Internal server error"
// @Security ApiKeyAuth
// @Router /user/balance/withdraw [post]
func (app *App) CreateWithdrawal(w http.ResponseWriter, r *http.Request) error {
	user := auth(w, r)
	if user == nil {
		return nil
	}

	var data withdrawalRequest
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

// ListWithdrawals godoc
// @Tags Withdrawals
// @Summary Get withdrawals
// @Description Get all withdrawals for current user
// @ID userListWithdrawals
// @Accept  json
// @Produce json
// @Success 200 {list} core.Withdrawal
// @Success 204 {empty}
// @Failure 401 {string} string "Not authenticated"
// @Failure 402 {string} string "User's balance exceeded"
// @Failure 500 {string} string "Internal server error"
// @Security ApiKeyAuth
// @Router /user/withdraws [get]
func (app *App) ListWithdrawals(w http.ResponseWriter, r *http.Request) error {
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
