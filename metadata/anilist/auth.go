package anilist

import (
	"context"
	"errors"

	"github.com/luevano/libmangal/metadata"
)

// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *Anilist) Authenticated() bool {
	if p.token == nil {
		return false
	}
	return p.token.AccessToken != ""
}

// Login authorizes an user with the given LoginOption.
func (p *Anilist) Login(ctx context.Context, loginOption metadata.LoginOption) error {
	switch loginOption := loginOption.(type) {
	case *metadata.CachedUserLoginOption:
		// TODO: check if the token is valid by re-fetching the user?
		// in which case, there is no need to cache the user
		// this currently assumes that both the token and user are valid
		p.token = loginOption.Token()
		p.user = loginOption.User
		return nil
	case *OAuthLoginOption:
		// Perform OAuth login (handles code and implicit grants)
		err := loginOption.authorize(ctx)
		if err != nil {
			return err
		}
		p.token = loginOption.Token()

		// Get authenticated user (this verifies the token)
		user, err := p.getAuthenticatedUser(ctx)
		if err != nil {
			// remove token as it's possible it's not valid
			p.token = nil
			return Error(err.Error())
		}
		p.user = user

		return nil
	default:
		return Error("unsuported login option " + loginOption.String())
	}
}

// Logout de-authorizes the currently authorized user.
func (p *Anilist) Logout() error {
	if !p.Authenticated() {
		return errors.New("no authenticated user to logout")
	}
	// To logout, removing the user and token is enough
	username := p.user.Name()
	p.user = nil
	p.token = nil
	p.logger.Log("user %q logged out of %q", username, p.Info().Name)
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
