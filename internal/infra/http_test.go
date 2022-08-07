package infra

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devsagul/gophemart/internal/action"
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
	err := action.UserRegister("bob", "sikret", app.repository.users, app.authBackend.GetAuthProvider(w))
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
		err = action.UserRegister("jane", "horse-correct", app.repository.users, app.authBackend.GetAuthProvider(w))
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
