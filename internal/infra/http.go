// Infra package incapsulates different things related to the http application,
// mainly application itself and handlers, also different utils

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

// default number of keys to be created on hydration
const NumKeysHydrated = 3

// custom handler function type
type Handler func(http.ResponseWriter, *http.Request) error

// Application
type App struct {
	// application router (http handler)
	Router        *chi.Mux
	store         storage.Storage
	accrualStream chan<- *core.Order
}

type userKey string

// context key for providing user
const UserKey = userKey("user")

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
			r := r.WithContext(context.WithValue(ctx, UserKey, user))
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

	userID, err := core.ParseToken(token, keys)

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

	user, err = app.store.ExtractUserByID(userID)

	switch err.(type) {
	case nil:
	case *storage.ErrKeyNotFound:
		return nil, nil
	case *storage.ErrUserNotFoundByID:
		return nil, nil
	default:
		return nil, err
	}

	return user, nil
}

// Hidrate keys of the application (create new HMAC keys if needed)
func (app *App) HydrateKeys() error {
	_, err := app.store.ExtractRandomKey()
	switch err.(type) {
	case *storage.ErrNoKeys:
		eg := errgroup.Group{}

		for i := 0; i <= NumKeysHydrated; i++ {
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

// Create new application
func NewApp(store storage.Storage, accrualStream chan<- *core.Order) *App {
	app := new(App)
	app.accrualStream = accrualStream
	app.store = store
	r := chi.NewRouter()
	app.Router = r

	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	r.Post("/api/user/register", app.newHandler(app.RegisterUser))
	r.Post("/api/user/login", app.newHandler(app.LoginUser))
	r.Post("/api/user/orders", app.newHandler(app.CreateOrder))
	r.Get("/api/user/orders", app.newHandler(app.ListOrders))
	r.Get("/api/user/balance", app.newHandler(app.GetBalance))
	r.Post("/api/user/balance/withdraw", app.newHandler(app.CreateWithdrawal))
	r.Get("/api/user/withdrawals", app.newHandler(app.ListWithdrawals))

	return app
}
