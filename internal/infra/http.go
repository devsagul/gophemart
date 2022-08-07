package infra

import "github.com/devsagul/gophemart/internal/auth"

type App struct {
	authBackend auth.AuthBackend
	repository  repository
}

func NewApp() *App {
	app := new(App)
	app.authBackend = auth.NoopAuthBackend{}
	app.repository = NewInMemoryRepository()

	return app
}
