package myanimelist

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/luevano/libmangal/metadata"
)

const (
	OAuthBaseURL      = "https://myanimelist.net/v1/oauth2/"
	OAuthTokenURL     = OAuthBaseURL + "token"
	OAuthAuthorizeURL = OAuthBaseURL + "authorize"
)

// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *MyAnimeList) Authenticated() bool {
	return p.token != ""
}

// Login authorizes an user with the given access token.
func (p *MyAnimeList) Login(ctx context.Context, token string) error {
	p.token = token

	user, err := p.getAuthenticatedUser(ctx)
	if err != nil {
		// remove token as it's possible it's not valid
		p.token = ""
		return Error(err.Error())
	}
	p.user = user

	return nil
}

// Logout de-authorizes the currently authorized user.
func (p *MyAnimeList) Logout() error {
	if !p.Authenticated() {
		return errors.New("no authenticated user to logout")
	}
	// To logout, removing the user and token is enough
	username := p.user.Name()
	p.user = nil
	p.token = ""
	p.logger.Log("user %q logged out of %q", username, p.Info().Name)
	return nil
}

// getAuthenticatedUser will query for the user data to the MyAnimeList API.
func (p *MyAnimeList) getAuthenticatedUser(ctx context.Context) (metadata.User, error) {
	params := url.Values{}
	params.Set("fields", userFields)

	var user *User
	err := p.request(ctx, http.MethodGet, "users/@me", params, p.commonMangaReqHeaders(), nil, &user)
	if err != nil {
		return nil, errors.New("getting authenticated user data: " + err.Error())
	}
	if user == nil {
		return nil, errors.New("received user data is nil")
	}

	return user, nil
}
