package infra

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// fixtures

func app(t *testing.T) (*App, *httptest.Server) {
	assert := assert.New(t)

	store := storage.NewMemStorage()

	app := NewApp(store, make(chan<- *core.Order, 255))

	server := httptest.NewServer(app.Router)
	err := app.HydrateKeys()
	if !assert.NoError(err) {
		assert.FailNow("could not setup application")
	}
	return app, server
}

func alice(t *testing.T, app *App) (*core.User, string) {
	assert := assert.New(t)

	alice, err := core.NewUser("alice", "correct-horse")
	if !assert.NoError(err) {
		assert.FailNow("could not create alice")
	}
	alice.Balance = decimal.New(1337, -2)
	err = app.store.CreateUser(alice)
	if !assert.NoError(err) {
		assert.FailNow("could not persist alice")
	}

	key, err := app.store.ExtractRandomKey()
	if !assert.NoError(err) {
		assert.FailNow("could not extract hmac key")
	}

	tokenAlice, err := core.GenerateToken(alice, key)
	if !assert.NoError(err) {
		assert.FailNow("could not generate token for alice")
	}

	authorizationHeaderAlice := fmt.Sprintf("Bearer %s", tokenAlice)
	return alice, authorizationHeaderAlice
}

func bob(t *testing.T, app *App) (*core.User, string) {
	assert := assert.New(t)

	bob, err := core.NewUser("bob", "sikret")
	if !assert.NoError(err) {
		assert.FailNow("could not create bob")
	}
	bob.Balance = decimal.New(420, 0)
	err = app.store.CreateUser(bob)
	if !assert.NoError(err) {
		assert.FailNow("could not persist bob")
	}

	key, err := app.store.ExtractRandomKey()
	if !assert.NoError(err) {
		assert.FailNow("could not extract hmac key")
	}

	tokenBob, err := core.GenerateToken(bob, key)
	if !assert.NoError(err) {
		assert.FailNow("could not generate token for bob")
	}

	authorizationHeaderBob := fmt.Sprintf("Bearer %s", tokenBob)
	return bob, authorizationHeaderBob
}

// tests
func TestRegisterUser(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/register"
	const content_type = "application/json"

	_, server := app(t)
	defer server.Close()

	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	type testCase struct {
		name         string
		body         string
		nilBody      bool
		expectedCode int
		xfail        bool
	}

	var testcases = []testCase{
		{
			"Register valid user",
			"{\"login\": \"alice\", \"password\": \"sikret\"}",
			false,
			http.StatusOK,
			false,
		},
		{
			"Register valid user second time",
			"{\"login\": \"alice\", \"password\": \"sikret\"}",
			false,
			http.StatusConflict,
			true,
		},
		{
			"Register user without login",
			"{\"password\": \"sikret\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Register user without password",
			"{\"login\": \"alice\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Register user with empty login",
			"{\"login\": \"\", \"password\": \"sikret\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Register user with empty password",
			"{\"login\": \"alice\", \"password\": \"\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Register user without request body",
			"",
			true,
			http.StatusBadRequest,
			true,
		},
	}
	for _, tCase := range testcases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			var body io.Reader
			if tCase.nilBody {
				body = nil
			} else {
				body = strings.NewReader(tCase.body)
			}
			resp, err := http.Post(url, content_type, body)
			if !assert.NoError(err) {
				return
			}

			authorizationHeader := resp.Header.Get("Authorization")

			if !tCase.xfail {
				assert.NotEmpty(authorizationHeader)
			}

			assert.Equal(tCase.expectedCode, resp.StatusCode)
		})
	}
}

func TestLoginUser(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/login"
	const content_type = "application/json"

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	user, err := core.NewUser("bob", "sikret")
	if !assert.NoError(t, err) {
		return
	}
	err = app.store.CreateUser(user)
	if !assert.NoError(t, err) {
		return
	}

	type testCase struct {
		name         string
		body         string
		nilBody      bool
		expectedCode int
		xfail        bool
	}

	var testCases = []testCase{
		{
			"Login existing user",
			"{\"login\": \"bob\", \"password\": \"sikret\"}",
			false,
			http.StatusOK,
			false,
		},
		{
			"Login existing user again",
			"{\"login\": \"bob\", \"password\": \"sikret\"}",
			false,
			http.StatusOK,
			false,
		},
		{
			"Login user without password",
			"{\"login\": \"bob\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Login user without login",
			"{\"password\": \"sikret\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Login user with empty password",
			"{\"login\": \"bob\", \"password\": \"\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Login user with empty login",
			"{\"login\": \"\", \"password\": \"sikret\"}",
			false,
			http.StatusBadRequest,
			true,
		},
		{
			"Login non-existing user",
			"{\"login\": \"eve\", \"password\": \"sikret\"}",
			false,
			http.StatusUnauthorized,
			true,
		},
		{
			"Login existing user with wrong password",
			"{\"login\": \"bob\", \"password\": \"qwerty\"}",
			false,
			http.StatusUnauthorized,
			true,
		},
		{
			"Login existing user without request body",
			"",
			true,
			http.StatusBadRequest,
			true,
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			var body io.Reader
			if tCase.nilBody {
				body = nil
			} else {
				body = strings.NewReader(tCase.body)
			}
			resp, err := http.Post(url, content_type, body)
			if !assert.NoError(err) {
				return
			}

			authorizationHeader := resp.Header.Get("Authorization")

			if !tCase.xfail {
				assert.NotEmpty(authorizationHeader)
			}

			assert.Equal(tCase.expectedCode, resp.StatusCode)
		})
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/orders"
	const content_type = "text/plain"
	const method = http.MethodPost

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	_, authorizationHeaderAlice := alice(t, app)
	_, authorizationHeaderBob := bob(t, app)

	client := http.Client{}

	type testCase struct {
		name         string
		body         string
		nilBody      bool
		expectedCode int
		auth         string
	}

	var testCases = []testCase{
		{
			"Upload order without authorization header set",
			"",
			true,
			http.StatusUnauthorized,
			"",
		},
		{
			"Upload order without body",
			"",
			true,
			http.StatusBadRequest,
			authorizationHeaderAlice,
		},
		{
			"Upload order with empty body",
			"",
			false,
			http.StatusBadRequest,
			authorizationHeaderAlice,
		},
		{
			"Upload valid order",
			"4561261212345467",
			false,
			http.StatusAccepted,
			authorizationHeaderAlice,
		},
		{
			"Upload valid order second time",
			"4561261212345467",
			false,
			http.StatusOK,
			authorizationHeaderAlice,
		},
		{
			"Upload confliction valid order",
			"4561261212345467",
			false,
			http.StatusConflict,
			authorizationHeaderBob,
		},
		{
			"Upload invalid order",
			"4561261212345463",
			false,
			http.StatusUnprocessableEntity,
			authorizationHeaderAlice,
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			var body io.Reader
			if tCase.nilBody {
				body = nil
			} else {
				body = strings.NewReader(tCase.body)
			}

			req, err := http.NewRequest(method, url, body)
			if !assert.NoError(err) {
				return
			}
			req.Header.Set("Authorization", tCase.auth)
			req.Header.Set("Content-Type", content_type)

			res, err := client.Do(req)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tCase.expectedCode, res.StatusCode)
		})
	}
}

func TestListOrders(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/orders"
	const method = http.MethodGet

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	_, authorizationHeaderAlice := alice(t, app)
	bob, authorizationHeaderBob := bob(t, app)

	secondsEastOfUTC := int((3 * time.Hour).Seconds())
	moscow := time.FixedZone("Moscow Time", secondsEastOfUTC)

	createdAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)

	order, err := core.NewOrder("4561261212345467", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateOrder(order)

	if !assert.NoError(t, err) {
		return
	}

	createdAt = time.Date(
		2022,
		time.August,
		8,
		21,
		40,
		0,
		0,
		moscow,
	)

	order, err = core.NewOrder("12345678903", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateOrder(order)
	if !assert.NoError(t, err) {
		return
	}

	client := http.Client{}

	type testCase struct {
		name         string
		auth         string
		expectedCode int
		checkBody    bool
		expectedBody string
	}

	var testCases = []testCase{
		{
			"Get orders unauthorized",
			"",
			http.StatusUnauthorized,
			false,
			"",
		},
		{
			"Get orders while none present",
			authorizationHeaderAlice,
			http.StatusNoContent,
			false,
			"",
		},
		{
			"Get orders while they are present",
			authorizationHeaderBob,
			http.StatusOK,
			true,
			"[{\"number\": \"12345678903\", \"status\": \"NEW\", \"uploaded_at\": \"2022-08-08T21:40:00+03:00\"}, {\"number\": \"4561261212345467\", \"status\": \"NEW\", \"uploaded_at\": \"2022-08-09T21:40:00+03:00\"}]",
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			req, err := http.NewRequest(method, url, nil)
			if !assert.NoError(err) {
				return
			}

			req.Header.Set("Authorization", tCase.auth)

			res, err := client.Do(req)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tCase.expectedCode, res.StatusCode)
			if tCase.checkBody {
				body := res.Body
				defer body.Close()
				bodyJSON, err := ioutil.ReadAll(body)
				if !assert.NoError(err) {
					return
				}
				assert.JSONEq(tCase.expectedBody, string(bodyJSON))
			}
		})
	}

}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/balance"
	const method = http.MethodGet

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	_, authorizationHeaderAlice := alice(t, app)
	bob, authorizationHeaderBob := bob(t, app)

	secondsEastOfUTC := int((3 * time.Hour).Seconds())
	moscow := time.FixedZone("Moscow Time", secondsEastOfUTC)

	createdAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)

	order, err := core.NewOrder("4561261212345467", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	if !assert.NoError(t, err) {
		return
	}

	processedAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)
	withdrawal, err := core.NewWithdrawal(order, decimal.New(25, -1), processedAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateWithdrawal(withdrawal, order)

	if !assert.NoError(t, err) {
		return
	}

	client := http.Client{}

	type testCase struct {
		name         string
		auth         string
		expectedCode int
		checkBody    bool
		expectedBody string
	}

	var testCases = []testCase{
		{
			"Get balance unauthorized",
			"",
			http.StatusUnauthorized,
			false,
			"",
		},
		{
			"Get balance without withdrawals",
			authorizationHeaderAlice,
			http.StatusOK,
			true,
			"{\"current\": 13.37, \"withdrawn\": 0}",
		},
		{
			"Get balance with withdrawals",
			authorizationHeaderBob,
			http.StatusOK,
			true,
			"{\"current\": 417.5, \"withdrawn\": 2.5}",
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			req, err := http.NewRequest(method, url, nil)
			if !assert.NoError(err) {
				return
			}

			req.Header.Set("Authorization", tCase.auth)

			res, err := client.Do(req)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tCase.expectedCode, res.StatusCode)
			if tCase.checkBody {
				body := res.Body
				defer body.Close()
				bodyJSON, err := ioutil.ReadAll(body)
				if !assert.NoError(err) {
					return
				}
				assert.JSONEq(tCase.expectedBody, string(bodyJSON))
			}
		})
	}
}

func TestCreateWithdrawal(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/balance/withdraw"
	const method = http.MethodPost

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	_, authorizationHeaderAlice := alice(t, app)
	bob, authorizationHeaderBob := bob(t, app)

	secondsEastOfUTC := int((3 * time.Hour).Seconds())
	moscow := time.FixedZone("Moscow Time", secondsEastOfUTC)

	createdAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)

	order, err := core.NewOrder("4561261212345467", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	processedAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)
	withdrawal, err := core.NewWithdrawal(order, decimal.New(25, -1), processedAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateWithdrawal(withdrawal, order)

	if !assert.NoError(t, err) {
		return
	}

	client := http.Client{}

	type testCase struct {
		name         string
		auth         string
		body         string
		nilBody      bool
		expectedCode int
		checkBody    bool
		expectedBody string
	}

	var testCases = []testCase{
		{
			"Create withdrawal unauthorized",
			"",
			"",
			false,
			http.StatusUnauthorized,
			false,
			"",
		},
		{
			"Create withdrawal with nil body",
			authorizationHeaderAlice,
			"",
			true,
			http.StatusBadRequest,
			false,
			"",
		},
		{
			"Create withdrawal without body",
			authorizationHeaderAlice,
			"",
			false,
			http.StatusBadRequest,
			false,
			"",
		},
		{
			"Create withdrawal without sum",
			authorizationHeaderAlice,
			"{\"order\": \"1337\"}",
			false,
			http.StatusBadRequest,
			false,
			"",
		},
		{
			"Create withdrawal without order",
			authorizationHeaderAlice,
			"{\"sum\": \"42\"}",
			false,
			http.StatusBadRequest,
			false,
			"",
		},
		{
			"Create withdrawal with invalid order",
			authorizationHeaderAlice,
			"{\"order\": \"1337\", \"sum\": \"42\"}",
			false,
			http.StatusUnprocessableEntity,
			false,
			"",
		},
		{
			"Create withdrawal with not enough balance",
			authorizationHeaderAlice,
			"{\"order\": \"2377225624\", \"sum\": \"42\"}",
			false,
			http.StatusPaymentRequired,
			false,
			"",
		},
		{
			"Create valid withdrawal",
			authorizationHeaderAlice,
			"{\"order\": \"2377225624\", \"sum\": \"1\"}",
			false,
			http.StatusOK,
			false,
			"",
		},
		{
			"Create withdrawal for existing order",
			authorizationHeaderAlice,
			"{\"order\": \"2377225624\", \"sum\": \"1\"}",
			false,
			http.StatusUnprocessableEntity,
			false,
			"",
		},
		{
			"Create withdrawal for existing order of other user",
			authorizationHeaderBob,
			"{\"order\": \"2377225624\", \"sum\": \"1\"}",
			false,
			http.StatusUnprocessableEntity,
			false,
			"",
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			var body io.Reader
			if tCase.nilBody {
				body = nil
			} else {
				body = strings.NewReader(tCase.body)
			}
			req, err := http.NewRequest(method, url, body)
			if !assert.NoError(err) {
				return
			}

			req.Header.Set("Authorization", tCase.auth)

			res, err := client.Do(req)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tCase.expectedCode, res.StatusCode)
			if tCase.checkBody {
				body := res.Body
				defer body.Close()
				bodyJSON, err := ioutil.ReadAll(body)
				if !assert.NoError(err) {
					return
				}
				assert.JSONEq(tCase.expectedBody, string(bodyJSON))
			}
		})
	}
}

func TestListWithdrawal(t *testing.T) {
	t.Parallel()

	const endpoint = "/api/user/withdrawals"
	const method = http.MethodGet

	app, server := app(t)

	defer server.Close()
	url := fmt.Sprintf("%s%s", server.URL, endpoint)

	_, authorizationHeaderAlice := alice(t, app)
	bob, authorizationHeaderBob := bob(t, app)

	secondsEastOfUTC := int((3 * time.Hour).Seconds())
	moscow := time.FixedZone("Moscow Time", secondsEastOfUTC)

	createdAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)

	order, err := core.NewOrder("4561261212345467", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	processedAt := time.Date(
		2022,
		time.August,
		9,
		21,
		40,
		0,
		0,
		moscow,
	)
	withdrawal, err := core.NewWithdrawal(order, decimal.New(25, -1), processedAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateWithdrawal(withdrawal, order)

	if !assert.NoError(t, err) {
		return
	}

	order, err = core.NewOrder("12345678903", bob, createdAt)

	if !assert.NoError(t, err) {
		return
	}

	processedAt = time.Date(
		2022,
		time.August,
		8,
		21,
		40,
		0,
		0,
		moscow,
	)
	withdrawal, err = core.NewWithdrawal(order, decimal.New(42, -1), processedAt)

	if !assert.NoError(t, err) {
		return
	}

	err = app.store.CreateWithdrawal(withdrawal, order)

	if !assert.NoError(t, err) {
		return
	}

	client := http.Client{}

	type testCase struct {
		name         string
		auth         string
		expectedCode int
		checkBody    bool
		expectedBody string
	}

	var testCases = []testCase{
		{
			"List withdrawals Create withdrawal unauthorized",
			"",
			http.StatusUnauthorized,
			false,
			"",
		},
		{
			"List withdrawals empty",
			authorizationHeaderAlice,
			http.StatusNoContent,
			false,
			"",
		},
		{
			"List withdrawals",
			authorizationHeaderBob,
			http.StatusOK,
			true,
			"[{\"order\": \"12345678903\", \"sum\": 4.2, \"processed_at\": \"2022-08-08T21:40:00+03:00\"}, {\"order\": \"4561261212345467\", \"sum\": 2.5, \"processed_at\": \"2022-08-09T21:40:00+03:00\"}]",
		},
	}

	for _, tCase := range testCases {
		t.Run(tCase.name, func(t *testing.T) {
			assert := assert.New(t)
			req, err := http.NewRequest(method, url, nil)
			if !assert.NoError(err) {
				return
			}

			req.Header.Set("Authorization", tCase.auth)

			res, err := client.Do(req)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tCase.expectedCode, res.StatusCode)
			if tCase.checkBody {
				body := res.Body
				defer body.Close()
				bodyJSON, err := ioutil.ReadAll(body)
				if !assert.NoError(err) {
					return
				}
				assert.JSONEq(tCase.expectedBody, string(bodyJSON))
			}
		})
	}
}
