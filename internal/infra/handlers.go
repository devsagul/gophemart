package infra

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/devsagul/gophemart/internal/action"
	"github.com/devsagul/gophemart/internal/storage"
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
	err = action.UserRegister(data.Login, data.Password, app.repository.users, app.auth.GetAuthProvider(w, r))
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
	err = action.UserLogin(data.Login, data.Password, app.repository.users, app.auth.GetAuthProvider(w, r))
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
	_, err := app.auth.GetAuthProvider(w, r).Auth()
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
}
