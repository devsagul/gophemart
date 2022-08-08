package infra

import "github.com/devsagul/gophemart/internal/auth"

type App struct {
	auth       auth.AuthBackend
	repository repository
}

func NewApp() *App {
	app := new(App)
	app.repository = NewInMemoryRepository()
	app.auth = auth.NewJwtAutAuthBackend(app.repository.users)
	return app
}
