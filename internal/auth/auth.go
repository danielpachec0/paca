package auth

import (
	"errors"
	"net/http"
)

func Auth(session map[string]int, cookies []*http.Cookie) (int, error) {
	var token string
	for _, cookie := range cookies {
		if cookie.Name == "_paca_token" {
			token = cookie.Value
		}
	}
	if token == "" {
		newError := errors.New("no authentication cookie provided")
		return 0, newError
	}
	id := session[token]
	if id == 0 {
		newError := errors.New("invalid token")
		return 0, newError
	}
	return id, nil
}
