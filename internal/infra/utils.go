package infra

import (
	"log"
	"net/http"

	"github.com/devsagul/gophemart/internal/core"
)

func auth(w http.ResponseWriter, r *http.Request) *core.User {
	ctx := r.Context()
	rawUser := ctx.Value("user")
	user, ok := rawUser.(*core.User)
	if !ok {
		user = nil
	}
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
	}
	return user
}

func wrapWrite(w http.ResponseWriter, body []byte) {
	_, err := w.Write(body)
	if err != nil {
		log.Printf("Error while writing response: %v", err)
	}
}
