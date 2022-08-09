package infra

import (
	"github.com/devsagul/gophemart/internal/auth"
	"github.com/devsagul/gophemart/internal/storage"
)

type App struct {
	auth  auth.AuthBackend
	store storage.Storage
}

func NewApp() *App {
	app := new(App)
	app.store = storage.NewMemStorage()
	app.auth = auth.NewJwtAutAuthBackend(app.store)
	return app
}
