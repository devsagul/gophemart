package infra

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devsagul/gophemart/internal/action"
	"github.com/devsagul/gophemart/internal/core"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name         string
	body         string
	expectedCode int
}

func TestRegisterUser(t *testing.T) {
	const URL = "/api/user/register"

	var testcases = []testCase{
		{
			"Register valid user",
			"{\"login\": \"alice\", \"password\": \"sikret\"}",
			http.StatusOK,
		},
		{
			"Register user without login",
			"{\"password\": \"sikret\"}",
			http.StatusBadRequest,
		},
		{
			"Register user without password",
			"{\"login\": \"alice\"}",
			http.StatusBadRequest,
		},
		{
			"Register user with empty login",
			"{\"login\": \"\", \"password\": \"sikret\"}",
			http.StatusBadRequest,
		},
		{
			"Register user user with empty password",
			"{\"login\": \"alice\", \"password\": \"\"}",
			http.StatusBadRequest,
		},
	}
	for _, tCase := range testcases {
		t.Run(tCase.name, func(t *testing.T) {
			app := NewApp()
			body := strings.NewReader(tCase.body)
			req := httptest.NewRequest(http.MethodPost, URL, body)
			w := httptest.NewRecorder()
			app.registerUser(w, req)

			assert.Equal(t, tCase.expectedCode, w.Code)
			// optional: check that user is authenticated
		})
	}

	t.Run("Register user without request body", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest(http.MethodPost, URL, nil)
		w := httptest.NewRecorder()
		app.registerUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Register valid user twice", func(t *testing.T) {
		app := NewApp()
		body := strings.NewReader("{\"login\": \"alice\", \"password\": \"sikret\"}")
		var teeBody bytes.Buffer
		req := httptest.NewRequest(http.MethodPost, URL, io.TeeReader(body, &teeBody))
		w := httptest.NewRecorder()
		app.registerUser(w, req)

		if w.Code != http.StatusOK {
			assert.FailNow(t, "could not register a user")
		}

		req = httptest.NewRequest(http.MethodPost, URL, &teeBody)
		w = httptest.NewRecorder()
		app.registerUser(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestLoginUser(t *testing.T) {
	const URL = "/app/user/login"
	app := NewApp()
	w := httptest.NewRecorder()
	err := action.UserRegister("bob", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
	if err != nil {
		assert.FailNow(t, "could not create user")
	}

	var testCases = []testCase{
		{
			"Login existing user",
			"{\"login\": \"bob\", \"password\": \"sikret\"}",
			http.StatusOK,
		},
		{
			"Login user without password",
			"{\"login\": \"bob\"}",
			http.StatusBadRequest,
		},
		{
			"Login user without login",
			"{\"password\": \"sikret\"}",
			http.StatusBadRequest,
		},
		{
			"Login user with empty password",
			"{\"login\": \"bob\", \"password\": \"\"}",
			http.StatusBadRequest,
		},
		{
			"Login user with empty login",
			"{\"login\": \"\", \"password\": \"sikret\"}",
			http.StatusBadRequest,
		},
		{
			"Login non-existing user",
			"{\"login\": \"eve\", \"password\": \"sikret\"}",
			http.StatusUnauthorized,
		},
		{
			"Login existing user with wrong password",
			"{\"login\": \"bob\", \"password\": \"qwerty\"}",
			http.StatusUnauthorized,
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			body := strings.NewReader(tCase.body)
			req := httptest.NewRequest(http.MethodPost, URL, body)
			w := httptest.NewRecorder()
			app.loginUser(w, req)

			assert.Equal(t, tCase.expectedCode, w.Code)
		})
	}

	t.Run("Login user without request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, URL, nil)
		w := httptest.NewRecorder()
		app.registerUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Login valid user twice", func(t *testing.T) {
		w := httptest.NewRecorder()
		err = action.UserRegister("jane", "horse-correct", app.store, app.auth.GetAuthProvider(w, nil))
		if err != nil {
			assert.FailNow(t, "could not create user")
		}

		body := strings.NewReader("{\"login\": \"jane\", \"password\": \"horse-correct\"}")
		var teeBody bytes.Buffer
		req := httptest.NewRequest(http.MethodPost, URL, io.TeeReader(body, &teeBody))
		w = httptest.NewRecorder()
		app.loginUser(w, req)

		if w.Code != http.StatusOK {
			assert.FailNow(t, "could not login user")
		}

		req = httptest.NewRequest(http.MethodPost, URL, &teeBody)
		w = httptest.NewRecorder()
		app.loginUser(w, req)

		if w.Code != http.StatusOK {
			assert.FailNow(t, "could not login user again")
		}
	})
}

func TestCreateOrder(t *testing.T) {
	const URL = "/api/user/orders"
	app := NewApp()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, URL, nil)
	app.createOrder(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	err := action.UserRegister("bob", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
	if err != nil {
		assert.FailNow(t, "could not create user")
	}
	authHeader := w.Result().Header.Get("Authorization")
	if authHeader == "" {
		assert.FailNow(t, "authorization header is not set")
	}

	t.Run("Upload order without body", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, nil)
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Upload order with empty body", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, strings.NewReader(""))
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Upload corect order", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, strings.NewReader("4561261212345467"))
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("Upload correct order second time", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, strings.NewReader("4561261212345467"))
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Upload correct order as another user", func(t *testing.T) {
		w = httptest.NewRecorder()
		err := action.UserRegister("eve", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
		if err != nil {
			assert.FailNow(t, "could not create user")
		}
		authHeader := w.Result().Header.Get("Authorization")
		if authHeader == "" {
			assert.FailNow(t, "authorization header is not set")
		}

		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, strings.NewReader("4561261212345467"))
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Upload incorrect order second time", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, URL, strings.NewReader("4561261212345468"))
		req.Header.Set("Authorization", authHeader)
		app.createOrder(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestListOrders(t *testing.T) {
	const URL = "/api/user/orders"
	app := NewApp()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, URL, nil)
	app.createOrder(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	err := action.UserRegister("bob", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
	if err != nil {
		assert.FailNow(t, "could not create user")
	}
	authHeader := w.Result().Header.Get("Authorization")
	if authHeader == "" {
		assert.FailNow(t, "authorization header is not set")
	}

	t.Run("Get orders while orders are not present", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, URL, nil)
		req.Header.Set("Authorization", authHeader)
		app.listOrders(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	createdAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		time.UTC,
	)

	user, err := app.store.ExtractUser("bob")
	if err != nil {
		assert.FailNow(t, "could not extract user")
	}
	order, err := core.NewOrder("4561261212345467", user, createdAt)
	expected := []core.Order{*order}
	err = app.store.CreateOrder(order)
	if err != nil {
		assert.FailNow(t, "could not create an order")
	}

	t.Run("Get orders", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, URL, nil)
		req.Header.Set("Authorization", authHeader)
		app.listOrders(w, req)
		var orders []core.Order
		body, err := ioutil.ReadAll(w.Body)
		if err != nil {
			assert.FailNow(t, "could not read response body")
		}
		json.Unmarshal(body, &orders)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, expected, orders)
	})
}

func TestGetBalance(t *testing.T) {
	const URL = "/api/user/balance/withdraw"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, URL, nil)

	app := NewApp()
	app.createOrder(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	err := action.UserRegister("bob", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
	if err != nil {
		assert.FailNow(t, "could not create user")
	}
	authHeader := w.Result().Header.Get("Authorization")
	if authHeader == "" {
		assert.FailNow(t, "authorization header is not set")
	}

	// get balance prior to withdrawal
	// set user balance and get it
	// create a withdrawal and get user balance
}

// create withdrawal

func TestCreateWithdrawal(t *testing.T) {
	const URL = "/api/user/withdrawals"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, URL, nil)

	app := NewApp()
	app.createOrder(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	err := action.UserRegister("bob", "sikret", app.store, app.auth.GetAuthProvider(w, nil))
	if err != nil {
		assert.FailNow(t, "could not create user")
	}
	authHeader := w.Result().Header.Get("Authorization")
	if authHeader == "" {
		assert.FailNow(t, "authorization header is not set")
	}

	t.Run("list withdrawals while they are empty", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, URL, nil)
		req.Header.Set("Authorization", authHeader)
		app.listWithdrawals(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	user, err := app.store.ExtractUser("bob")
	if err != nil {
		assert.FailNow(t, "could not extract user")
	}
	user.Balance = decimal.New(420, 0)
	err = app.store.PersistUser(user)
	if err != nil {
		assert.FailNow(t, "could not update user's balance")
	}

	order, err := core.NewOrder("4561261212345467", user, time.Now())
	withdrawal, err := core.NewWithdrawal(order, decimal.New(13, 37), time.Now())
	exp := []core.Withdrawal{*withdrawal}
	expected, err := json.Marshal(exp)
	if err != nil {
		assert.FailNow(t, "could not mashal withdrawals")
	}
	err = app.store.CreateWithdrawal(withdrawal, order)
	if err != nil {
		assert.FailNow(t, "could not create a withdrawal")
	}

	t.Run("Create withdrawal", func(t *testing.T) {
		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, URL, nil)
		req.Header.Set("Authorization", authHeader)
		app.listWithdrawals(w, req)
		body, err := ioutil.ReadAll(w.Body)
		if err != nil {
			assert.FailNow(t, "could not read response body")
		}
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, expected, body)
	})
}
