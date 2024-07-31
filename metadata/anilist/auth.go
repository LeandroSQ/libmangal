package anilist

import (
	"context"
	"errors"

	"github.com/luevano/libmangal/metadata"
)

const (
	OAuthBaseURL      = "https://anilist.co/api/v2/oauth/"
	OAuthPinURL       = OAuthBaseURL + "pin"
	OAuthTokenURL     = OAuthBaseURL + "token"
	OAuthAuthorizeURL = OAuthBaseURL + "authorize"
)

// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *Anilist) Authenticated() bool {
	return p.token != ""
}

// Login authorizes an user with the given access token.
func (p *Anilist) Login(ctx context.Context, token string) error {
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
func (p *Anilist) Logout() error {
	if !p.Authenticated() {
		return errors.New("no authenticated user to logout")
	}
	p.user = nil
	p.token = ""
	return nil
}

// getAuthenticatedUser will query for the user data to the Anilist API.
func (p *Anilist) getAuthenticatedUser(ctx context.Context) (metadata.User, error) {
	body := apiRequestBody{
		Query: queryViewer,
	}
	data, err := sendRequest[userData](ctx, p, body)
	if err != nil {
		return nil, errors.New("getting authenticated user data: " + err.Error())
	}
	if data.Viewer == nil {
		return nil, errors.New("received user data is nil")
	}
	return data.Viewer, nil
}
