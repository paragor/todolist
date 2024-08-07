package httpserver

import (
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
)

type AuthChainConfig struct {
	*AuthBaseConfig
	*AuthTelegramConfig
	*AuthTokenConfig
	*AuthOidcConfig
}

func (h *httpServer) AuthChainMiddleware() mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if h.authConfig.AuthTelegramConfig != nil && isAuthorizedByTelegram(request, *h.authConfig.AuthTelegramConfig) {
				handler.ServeHTTP(writer, request)
				return
			}
			if h.authConfig.AuthTokenConfig != nil && isAuthorizedByToken(request, *h.authConfig.AuthTokenConfig) {
				handler.ServeHTTP(writer, request)
				return
			}
			if h.authConfig.AuthBaseConfig != nil {
				if isAuthorizedByBaseAuth(request, *h.authConfig.AuthBaseConfig) {
					handler.ServeHTTP(writer, request)
					return
				}
				if isRequireForceBaseAuth(request) {
					writer.Header().Set("WWW-Authenticate", `Basic realm="todolist"`)
					writeHtmx(writer, "component/auth_base_challenge", "", http.StatusUnauthorized)
					return
				}
			}
			if h.oidc != nil && h.oidc.isAuthorizedByOidc(request) {
				handler.ServeHTTP(writer, request)
				return
			}
			h.htmxPageLogin(writer, request)
		})
	}
}

func (h *httpServer) htmxPageLogin(writer http.ResponseWriter, request *http.Request) {
	authContext := ""
	if h.authConfig != nil {
		if h.authConfig.AuthOidcConfig != nil {
			oidcHtmx, deferFn1, err := renderHtmx("component/auth_oidc_challenge", "")
			defer deferFn1()
			if err != nil {
				http.Error(writer, "error on render oidc auth", 500)
				return
			}
			authContext += oidcHtmx.String()
		}
		if h.authConfig.AuthTelegramConfig != nil {
			telegramHtmx, deferFn1, err := renderHtmx("component/auth_telegram_challenge", "")
			defer deferFn1()
			if err != nil {
				http.Error(writer, "error on render telegram auth", 500)
				return
			}
			authContext += telegramHtmx.String()
		}
		if h.authConfig.AuthBaseConfig != nil {
			baseAuthHtmx, deferFn2, err := renderHtmx("component/auth_base_challenge", "")
			defer deferFn2()
			if err != nil {
				http.Error(writer, "error on render base auth", 500)
				return
			}
			authContext += baseAuthHtmx.String()
		}
	}
	writeHtmx(writer, "page/index", template.HTML(authContext), 403)
}
