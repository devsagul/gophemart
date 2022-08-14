package infra

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v4"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

const NUM_KEYS_HYDRATED = 3

type Handler func(http.ResponseWriter, *http.Request) error

type App struct {
	store  storage.Storage
	Router *chi.Mux
}

func (app *App) newHandler(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errChan := make(chan error)
		ctx := r.Context()

		go func() {
			user, err := app.authenticate(r)
			if err != nil {
				errChan <- err
				return
			}
			r := r.WithContext(context.WithValue(ctx, "user", user))
			errChan <- h(w, r)
		}()

		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("Unhandled error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				wrapWrite(w, []byte("{\"status\": \"error\", \"message\": \"Internal server error\"}"))
			}
		case <-ctx.Done():
		}
	}
}

func (app *App) authenticate(r *http.Request) (*core.User, error) {
	var user *core.User = nil
	header := r.Header.Get("Authorization")

	var token string
	_, err := fmt.Fscanf(strings.NewReader(header), "Bearer %s", &token)
	if err != nil {
		return nil, nil
	}

	keys, err := app.store.ExtractAllKeys()

	if err != nil {
		return nil, err
	}

	userId, err := core.ParseToken(token, keys)

	switch err.(type) {
	case nil:
	case *jwt.ValidationError:
		return nil, nil
	case *core.ErrExpiredToken:
		return nil, nil
	case *core.ErrUnexpectedSigningMethod:
		return nil, nil
	default:
		return nil, err
	}
	if err != nil {
		return nil, nil
	}

	user, err = app.store.ExtractUserById(userId)

	switch err.(type) {
	case nil:
	case *storage.ErrKeyNotFound:
		return nil, nil
	case *storage.ErrUserNotFoundById:
		return nil, nil
	default:
		return nil, err
	}

	return user, nil
}

func (app *App) HydrateKeys() error {
	_, err := app.store.ExtractRandomKey()
	switch err.(type) {
	case *storage.ErrNoKeys:
		eg := errgroup.Group{}

		for i := 0; i <= NUM_KEYS_HYDRATED; i++ {
			eg.Go(func() error {
				key, err := core.NewKey()
				if err != nil {
					return err
				}
				return app.store.CreateKey(key)
			})
		}

		return eg.Wait()
	default:
		return err
	}
}

func (app *App) login(user *core.User, w http.ResponseWriter) error {
	key, err := app.store.ExtractRandomKey()
	if err != nil {
		return err
	}

	token, err := core.GenerateToken(user, key)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Bearer %s", token)
	w.Header().Set("Authorization", header)
	return nil
}

func NewApp(store storage.Storage) *App {
	app := new(App)
	app.store = store
	r := chi.NewRouter()
	app.Router = r

	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	r.Post("/api/user/register", app.newHandler(app.registerUser))
	r.Post("/api/user/login", app.newHandler(app.loginUser))
	r.Post("/api/user/orders", app.newHandler(app.createOrder))
	r.Get("/api/user/orders", app.newHandler(app.listOrders))
	r.Get("/api/user/balance", app.newHandler(app.getBalance))
	r.Post("/api/user/balance/withdraw", app.newHandler(app.createWithdrawal))
	r.Get("/api/user/withdrawals", app.newHandler(app.listWithdrawals))

	return app
}
