package anilist

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/luevano/libmangal/metadata"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"
)

const (
	OAuthBaseURL      = "https://anilist.co/api/v2/oauth/"
	OAuthPinURL       = OAuthBaseURL + "pin"
	OAuthTokenURL     = OAuthBaseURL + "token"
	OAuthAuthorizeURL = OAuthBaseURL + "authorize"

	// TODO: make configurable (and used in the oauth handlers)
	OAuthServerBaseRUL     = "http://localhost:6969/oauth/al/"
	OAuthServerLoginURL    = OAuthServerBaseRUL + "login"
	OAuthServerCallbackURL = OAuthServerBaseRUL + "callback"

	// should only be used on implicit grant due to
	// access token being part of the fragment
	resendFragmentAsParam = `
<html>
<head>
</head>
<body onload="resend()">

<script>
  function resend(){
    var hash = window.location.hash.substring(1);
    window.location.href = '?' + hash
  }
</script>

</body>
</html>
`
)

var OAuthEndpoint = oauth2.Endpoint{
	AuthURL:   OAuthAuthorizeURL,
	TokenURL:  OAuthTokenURL,
	AuthStyle: oauth2.AuthStyleInParams,
}

var _ metadata.LoginOption = (*OAuthLoginOption)(nil)

// WARN: the implicit grant (empty ClientSecret) should be used
// with caution and only if completely needed, due to the access
// token being resent (to the same callback URL) on the URL params.
// Anilist api was designed to send it as part of the URL fragment,
// specifically so that servers didn't have access to the token directly.
//
// OAuthLoginOption is an implementation of metadata.LoginOption
// that handles OAuth2 login.
type OAuthLoginOption struct {
	// ClientID is the application's ID.
	//
	// Must be non-empty.
	ClientID string

	// ClientSecret is the application's secret.
	//
	// When empty, the OAuth is an implicit grant.
	ClientSecret string

	// code is the access code; when the ClientSecret
	// is empty, this is the access token.
	code  string
	token *oauth2.Token
}

func NewOAuthLoginOption(clientID, clientSecret string) (*OAuthLoginOption, error) {
	if clientID == "" {
		return nil, OAuthErrEmptyClientID
	}

	return &OAuthLoginOption{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, nil
}

// getOAuthConfig is a convenience method to get the oauth2 Config for
// this OAuthLoginOption.
func (o *OAuthLoginOption) getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		Endpoint:     OAuthEndpoint,
		RedirectURL:  OAuthServerCallbackURL, // TODO: make configurable?
	}
}

// getRequestBody is a convenience method to get the OAuthLoginOption
// as a code grant request body.
//
// Only needed when there is a ClientSecret.
func (o *OAuthLoginOption) getRequestBody() map[string]string {
	return map[string]string{
		"client_id":     o.ClientID,
		"client_secret": o.ClientSecret,
		"code":          o.code,
		"grant_type":    "authorization_code",
		"redirect_uri":  OAuthServerCallbackURL, // TODO: make configurable?
	}
}

// getImplicitGrantURL is a convenience method to get the implicit grant
// oauth url, as it requires the response_type to be 'token' and not include
// the redirect uri.
func (o *OAuthLoginOption) getImplicitGrantURL() string {
	var buf bytes.Buffer
	c := o.getOAuthConfig()
	buf.WriteString(c.Endpoint.AuthURL)
	v := url.Values{
		"response_type": {"token"}, // for implicit, needs to be token
		"client_id":     {o.ClientID},
	}
	if strings.Contains(c.Endpoint.AuthURL, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

// String the name of the login option, for logging purposes.
func (o *OAuthLoginOption) String() string {
	return "Anilist OAuth2"
}

// Token returns the authorization token (useful for caching).
func (o *OAuthLoginOption) Token() *oauth2.Token {
	return o.token
}

// authorize will perform the OAuth authorization procedure by:
//
// 1. Starting an http server to handle anilist callbacks with the
// access code/token. For more:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/getting-started#auth-pin
//
// 2. If doing a code grant (ClientSecret is not empty), converting
// the acces code to an access token. For more:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant#converting-authorization-codes-to-access-tokens
//
// 3. Retrieving (or generating in case of implicit grant) the oauth2 Token.
//
// The http server lives for a max duration of 1 minute or until the access
// token is successfully retrieved/built.
func (o *OAuthLoginOption) authorize(ctx context.Context) (tokenErr error) {
	if o.ClientID == "" {
		return OAuthErrEmptyClientID
	}

	// TODO: make timeout duration configurable?
	//
	// a new ctx is required to close the http server
	// after some time or after receiving the code
	srvCtx, srvCtxCancel := context.WithTimeout(ctx, time.Minute)

	// get the correct handler depending on the grant type
	var handler http.Handler
	if o.ClientSecret != "" {
		handler = o.codeGrantHandler(srvCtxCancel, &tokenErr)
	} else {
		handler = o.implicitGrantHandler(srvCtxCancel, &tokenErr)
	}

	// start a new http server
	s := &http.Server{
		Addr:    ":6969", // TODO: make configurable
		Handler: handler,
	}
	go func() {
		s.ListenAndServe()
	}()

	// open the web browser on the login url
	if err := open.Start(OAuthServerLoginURL); err != nil {
		return err
	}

	// wait until the server ctx timeout runs out or
	// the access code is retrieved to close the server
	<-srvCtx.Done()
	if err := s.Shutdown(ctx); err != nil {
		return err
	}

	// check if there was an error during
	// the token retrieval or if nil
	if tokenErr != nil {
		return tokenErr
	}
	if o.token == nil {
		return OAuthErrNilToken
	}
	return nil
}

// codeGrantHandler provides the http handlers when
// using a code grant oauth (ClientSecret is non-empty)
func (o *OAuthLoginOption) codeGrantHandler(ctxCancel context.CancelFunc, tokenErr *error) http.Handler {
	callbackCount := 0
	closeMsg := "; this window can now be closed"
	// TODO: don't hardcode the server paths
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/al/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, o.getOAuthConfig().AuthCodeURL(""), http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/oauth/al/callback", func(w http.ResponseWriter, r *http.Request) {
		callbackCount++
		if callbackCount > 1 {
			*tokenErr = OAuthErrTooManyCallbacks
			w.Write([]byte("error: " + (*tokenErr).Error() + closeMsg))
			ctxCancel()
			return
		}

		o.code = r.FormValue("code")
		if o.code == "" {
			*tokenErr = OAuthErrEmptyCode
			w.Write([]byte("error: " + (*tokenErr).Error() + closeMsg))
			ctxCancel()
			return
		}

		token, err := o.getOAuthConfig().Exchange(context.Background(), o.code)
		if err != nil {
			*tokenErr = err
			w.Write([]byte("error: " + (*tokenErr).Error() + closeMsg))
			ctxCancel()
			return
		}
		o.token = token

		w.Write([]byte("successfully got access token (code grant)" + closeMsg))
		ctxCancel()
	})
	return mux
}

// implicitGrantHandler provides the http handlers when
// using an implicit grant oauth (ClientSecret is empty)
func (o *OAuthLoginOption) implicitGrantHandler(ctxCancel context.CancelFunc, tokenErr *error) http.Handler {
	callbackCount := 0
	closeMsg := "; this window can now be closed"
	// TODO: don't hardcode the server paths
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/al/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, o.getImplicitGrantURL(), http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/oauth/al/callback", func(w http.ResponseWriter, r *http.Request) {
		// WARN: the token is sent from anilist as part of the url fragment,
		// so to be able to get the access token, the fragment needs to be
		// "intercepted" with javascript, and then resent as part of the url
		// params; this handles that part, it should only run once
		if r.URL.RawQuery == "" {
			callbackCount++
			if callbackCount > 1 {
				*tokenErr = OAuthErrTooManyCallbacks
				w.Write([]byte("error: " + (*tokenErr).Error() + closeMsg))
				ctxCancel()
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, resendFragmentAsParam)
			return
		}

		o.code = r.FormValue("access_token")
		if o.code == "" {
			*tokenErr = OAuthErrEmptyCode
			w.Write([]byte("error: " + (*tokenErr).Error() + closeMsg))
			ctxCancel()
			return
		}

		// build the default token with the year expiry
		o.token = &oauth2.Token{
			AccessToken: o.code,
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(31536000 * time.Second),
		}
		w.Write([]byte("successfully got access token (implicit grant)" + closeMsg))
		ctxCancel()
		return
	})
	return mux
}
