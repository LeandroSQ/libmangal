package myanimelist

import (
	"context"
	"errors"
)

const (
	OAuthBaseURL      = "https://myanimelist.net/v1/oauth2/"
	OAuthTokenURL     = OAuthBaseURL + "token"
	OAuthAuthorizeURL = OAuthBaseURL + "authorize"
)

// FIX: implement
//
// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *MyAnimeList) Authenticated() bool {
	return false
}

// FIX: implement
//
// Login authorizes an user with the given access token.
func (p *MyAnimeList) Login(ctx context.Context, token string) error {
	return errors.ErrUnsupported
}

// FIX: implement
//
// Logout de-authorizes the currently authorized user.
func (p *MyAnimeList) Logout() error {
	return errors.ErrUnsupported
}
