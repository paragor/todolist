package httpserver

import (
	"net/http"
)

type AuthTokenConfig struct {
	Token string
}

func isAuthorizedByToken(request *http.Request, cfg AuthTokenConfig) bool {
	return request.Header.Get("Authorization") == cfg.Token
}
