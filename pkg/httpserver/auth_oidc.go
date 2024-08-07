package httpserver

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"net/http"
	"time"
)

const oidcCookie = "oidc_cookie"

type AuthOidcConfig struct {
	ClientId        string
	ClientSecret    string
	IssuerUrl       string
	CookieKey       string
	Scopes          []string
	WhitelistEmails []string
}

type authOidcContext struct {
	cfg                    AuthOidcConfig
	provider               rp.RelyingParty
	successfulRedirectPath string
	idTokenCookieName      string
}

func newOidcContext(cfg AuthOidcConfig, callbackUrl string, successfulRedirectPath string) (*authOidcContext, error) {
	cookieHandler := httphelper.NewCookieHandler([]byte(cfg.CookieKey), []byte(cfg.CookieKey))
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	provider, err := rp.NewRelyingPartyOIDC(ctx, cfg.IssuerUrl, cfg.ClientId, cfg.ClientSecret, callbackUrl, cfg.Scopes, options...)
	if err != nil {
		return nil, fmt.Errorf("error creating provider %v", err)
	}
	return &authOidcContext{
		cfg:                    cfg,
		provider:               provider,
		idTokenCookieName:      "oidc_id_token",
		successfulRedirectPath: successfulRedirectPath,
	}, nil
}

func (oc *authOidcContext) isAuthorizedByOidc(request *http.Request) bool {
	idToken, err := oc.provider.CookieHandler().CheckCookie(request, oc.idTokenCookieName)
	if err != nil || idToken == "" {
		return false
	}
	claim, err := rp.VerifyIDToken[*oidc.IDTokenClaims](request.Context(), idToken, oc.provider.IDTokenVerifier())
	if err != nil {
		return false
	}
	return oc.isValidEmail(claim.UserInfoEmail)
}

func (oc *authOidcContext) isValidEmail(info oidc.UserInfoEmail) bool {
	for _, validEmail := range oc.cfg.WhitelistEmails {
		if validEmail == info.Email && info.EmailVerified {
			return true
		}
	}
	return false
}
func (oc *authOidcContext) userInfoCallback(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, rp rp.RelyingParty, info *oidc.UserInfo) {
	if len(oc.cfg.WhitelistEmails) > 0 {
		if !oc.isValidEmail(info.UserInfoEmail) {
			http.Error(w, "email is blocked", http.StatusUnauthorized)
			return
		}
	}
	if err := oc.provider.CookieHandler().SetCookie(w, oc.idTokenCookieName, tokens.IDToken); err != nil {
		http.Error(w, "cant set cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, oc.successfulRedirectPath, http.StatusFound)
}

func (oc *authOidcContext) AuthCallbackHandler() http.Handler {
	return rp.CodeExchangeHandler(rp.UserinfoCallback(oc.userInfoCallback), oc.provider)
}

func (oc *authOidcContext) AuthLoginHandler() http.Handler {
	return rp.AuthURLHandler(func() string { return uuid.New().String() }, oc.provider)
}
