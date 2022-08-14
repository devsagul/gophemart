package auth

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

const (
	SESSION_ID_LENGTH = 64
	SESSION_LIFETIME  = 24 * 60 * 60 * time.Second
)

func NewSession(login string) (*Session, error) {
	randID := make([]byte, SESSION_ID_LENGTH)
	_, err := io.ReadFull(rand.Reader, randID)
	if err != nil {
		return nil, err
	}

	sessionID := base64.RawStdEncoding.EncodeToString(randID)

	return &Session{
		sessionID,
		login,
		time.Now().Add(SESSION_LIFETIME),
		true,
	}, nil
}

func RefreshSession(s *Session) (*Session, error) {
	timeLeft := time.Now().Sub(s.expiry)
	if 2*timeLeft > SESSION_LIFETIME {
		return s, nil
	}
	s.active = false
	return NewSession(s.username)
}

func (s *Session) Cookie() http.Cookie {
	return http.Cookie{
		Name:    "Session",
		Value:   s.ID,
		Path:    "/",
		Expires: s.expiry,
	}
}
