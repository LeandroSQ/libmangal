package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/luevano/libmangal/metadata"
)

const (
	DefaultOAuthRedirectURI = "https://anilist.co/api/v2/oauth/pin"
	oAuthTokenURL           = "https://anilist.co/api/v2/oauth/token"
	oAuthGetTokenURL        = "https://anilist.co/api/v2/oauth/authorize?client_id=%s&response_type=token"
)

// Authenticated returns true if there is a currently authenticated
// user (there exists an available access token and user data).
func (p *Anilist) Authenticated() bool {
	return p.authData.AccessToken != ""
}

// Login authorizes an user with the given credentials.
func (p *Anilist) Login(ctx context.Context, credentials metadata.CodeGrant) (metadata.User, metadata.AuthData, error) {
	if err := credentials.Validate(); err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}

	if credentials.ClientSecret != "" {
		return p.AuthorizeWithCodeGrant(ctx, credentials)
	}

	return p.AuthorizeWithAccessToken(ctx, credentials.Code)
}

// Logout de-authorizes the currently authorized user.
func (p *Anilist) Logout() error {
	if !p.Authenticated() {
		return errors.New("no authenticated user to logout")
	}
	// To logout, removing the user and token is enough
	username := p.user.Name()
	p.user = nil
	p.authData = metadata.AuthData{}
	p.logger.Log("user %q logged out of anilist", username)
	return nil
}

// FIX: move to provider.go
//
// DeleteCachedUser will delete the specified user cached access token and data.
func (p *Anilist) DeleteCachedUser(username string) error {
	p.logger.Log("deleting cached authentication user data for %q", username)
	err := p.store.deleteUser(username)
	if err != nil {
		return err
	}
	err = p.store.deleteAuthData(username)
	if err != nil {
		return err
	}
	return nil
}

// FIX: move to provider.go
//
// AuthorizeCachedUser will try to get the cached authentication data for the given
// username. If the data exists, Anilist will be authenticated with this user.
func (p *Anilist) AuthorizeCachedUser(username string) (bool, error) {
	p.logger.Log("authenticating Anilist via cached user %q", username)
	// Get stored authData
	authData, found, err := p.store.getAuthData(username)
	if err != nil {
		return false, Error(err.Error())
	}
	if !found {
		return false, nil
	}
	p.authData = authData

	// Get stored user (if there is a token, there should be an user)
	user, found, err := p.store.getUser(username)
	if err != nil {
		return false, Error(err.Error())
	}
	// If no user found, delete the found access token
	if !found {
		p.logger.Log("cached access token for user %q found but there is no cached user data, need to re-authenticate", username)
		p.authData = metadata.AuthData{}
		err := p.store.deleteAuthData(username)
		if err != nil {
			return false, Error(err.Error())
		}
		return false, nil
	}
	p.user = user
	return true, nil
}

// AuthorizeWithCodeGrant will authorize the client (login)
// via code grant, as specified in:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant
//
// When a client is authorized all API requests will
// have the access token attached (will be authorized).
func (p *Anilist) AuthorizeWithCodeGrant(ctx context.Context, codeGrant metadata.CodeGrant) (metadata.User, metadata.AuthData, error) {
	p.logger.Log("authenticating Anilist via code grant")
	// Access authData
	authData, err := p.getTokenFromCode(ctx, codeGrant)
	if err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	p.authData = authData

	// Get authenticated user (this verifies the token)
	user, err := p.getAuthenticatedUser(ctx)
	if err != nil {
		// remove token as it's possible it's not valid
		p.authData = metadata.AuthData{}
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	p.user = user

	// Store the user and token to cache
	if err := p.store.setUser(user.Name(), user); err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	if err := p.store.setAuthData(user.Name(), authData); err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	return user, authData, nil
}

// AuthorizeWithAccessToken will authorize the client (login)
// via a pre-obtained access token, as specified in:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/implicit-grant
//
// When a client is authorized all API requests will
// have the access token attached (will be authorized).
func (p *Anilist) AuthorizeWithAccessToken(ctx context.Context, token string) (metadata.User, metadata.AuthData, error) {
	p.logger.Log("authenticating Anilist via acces token")
	// build a basic auth data obj (lacking refresh token)
	authData := metadata.AuthData{
		AccessToken: token,
		CreatedAt:   int(time.Now().Unix()),
		ExpiresIn:   31536000, // the same default that anilist gives (1 yr)
		TokenType:   "Bearer",
	}
	p.authData = authData

	// Get authenticated user (this verifies the token)
	user, err := p.getAuthenticatedUser(ctx)
	if err != nil {
		// remove token as it's possible it's not valid
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	p.user = user

	// Store the user and token to cache
	if err := p.store.setUser(user.Name(), user); err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}
	if err := p.store.setAuthData(user.Name(), authData); err != nil {
		return nil, metadata.AuthData{}, Error(err.Error())
	}

	return p.user, authData, nil
}

// getTokenFromCode will convert the authorization code into an AccessToken.
func (p *Anilist) getTokenFromCode(ctx context.Context, codeGrant metadata.CodeGrant) (metadata.AuthData, error) {
	// prepare the request body
	if err := codeGrant.Validate(); err != nil {
		return metadata.AuthData{}, err
	}
	reqBody := codeGrant.ToReqBody()
	if codeGrant.RedirectURI == "" {
		reqBody["redirect_uri"] = DefaultOAuthRedirectURI
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return metadata.AuthData{}, err
	}

	resp, err := p.genericRequest(ctx, http.MethodPost, oAuthTokenURL, bytes.NewBuffer(body), false)
	if err != nil {
		return metadata.AuthData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return metadata.AuthData{}, errors.New("non-OK status response code: " + resp.Status)
	}

	var authData metadata.AuthData
	err = json.NewDecoder(resp.Body).Decode(&authData)
	if err != nil {
		return metadata.AuthData{}, err
	}
	// Anilist doesn't provide created_at field
	// (not that it matters as it expires in a year anyways)
	authData.CreatedAt = int(time.Now().Unix())

	return authData, nil
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
