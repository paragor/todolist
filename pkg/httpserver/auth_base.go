package httpserver

import (
	"net/http"
)

type AuthBaseConfig struct {
	Login    string
	Password string
}

func isAuthorizedByBaseAuth(request *http.Request, cfg AuthBaseConfig) bool {
	inUser, inPassword, ok := request.BasicAuth()
	return ok && inUser == cfg.Login && inPassword == cfg.Password
}
func isRequireForceBaseAuth(request *http.Request) bool {
	cookie, err := request.Cookie("base_auth_challenge")
	if err != nil {
		return false
	}
	return cookie.Value == "true"
}
