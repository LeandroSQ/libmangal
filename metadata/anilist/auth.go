package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

const (
	oAuthPinURL      = "https://anilist.co/api/v2/oauth/pin"
	oAuthTokenURL    = "https://anilist.co/api/v2/oauth/token"
	oAuthGetTokenURL = "https://anilist.co/api/v2/oauth/authorize?client_id=%s&response_type=token"
)

// CodeGrant is used to authenticate with a given client,
// which can be used when the client can store the ID and Secret securely,
// (meaning that the end user doesn't have access to these).
//
// This should be used when using your own application. For more:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/getting-started#using-oauth,
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant
type CodeGrant struct {
	// ID is the application client ID.
	ID string

	// Secret is the application client secret.
	Secret string

	// Code is the user provided access code. One time use.
	Code string

	// RedirectURI callback to receive the access code.
	//
	// Defaults to https://anilist.co/api/v2/oauth/pin,
	// which needs to be manually copied.
	RedirectURI string
}

func (c CodeGrant) Validate() error {
	if c.ID == "" {
		return CodeGrantError("ID is empty")
	}
	if c.Secret == "" {
		return CodeGrantError("Secret is empty")
	}
	if c.Code == "" {
		return CodeGrantError("Code is empty")
	}
	return nil
}

func (c CodeGrant) reqBody() map[string]string {
	uri := c.RedirectURI
	if uri == "" {
		uri = oAuthPinURL
	}
	return map[string]string{
		"client_id":     c.ID,
		"client_secret": c.Secret,
		"code":          c.Code,
		"grant_type":    "authorization_code",
		"redirect_uri":  uri,
	}
}

// Logout of the authenticated user account.
//
// Returns an error if there is no authenticated user to be logged out.
func (a *Anilist) Logout(deleteCache bool) error {
	if !a.Authenticated() {
		return AuthError("no authenticated user to logout")
	}
	// To loggout, removing the user and token is enough
	username := a.user.Name
	a.user = User{}
	a.token = ""
	a.logger.Log("user %q logged out of anilist", username)

	if deleteCache {
		return a.DeleteCachedUser(username)
	}
	return nil
}

// DeleteCachedUser will delete the specified user cached access token and data.
func (a *Anilist) DeleteCachedUser(username string) error {
	a.logger.Log("deleting cached authentication user data for %q", username)
	err := a.store.deleteUser(username)
	if err != nil {
		return err
	}
	err = a.store.deleteAuthToken(username)
	if err != nil {
		return err
	}
	return nil
}

// AuthorizeCachedUser will try to get the cached authentication data for the given
// username. If the data exists, Anilist will be authenticated with this user.
func (a *Anilist) AuthorizeCachedUser(username string) (bool, error) {
	a.logger.Log("authenticating Anilist via cached user %q", username)
	// Get stored token
	token, found, err := a.store.getAuthToken(username)
	if err != nil {
		return false, AuthError(err.Error())
	}
	if !found {
		return false, nil
	}
	a.token = token

	// Get stored user (if there is a token, there should be an user)
	user, found, err := a.store.getUser(username)
	if err != nil {
		return false, AuthError(err.Error())
	}
	// If no user found, delete the found access token
	if !found {
		a.logger.Log("cached access token for user %q found but there is no cached user data, need to re-authenticate", username)
		a.token = ""
		err := a.store.deleteAuthToken(username)
		if err != nil {
			return false, AuthError(err.Error())
		}
		return false, nil
	}
	a.user = user
	return true, nil
}

// AuthorizeWithCodeGrant will authorize the client (login)
// via code grant, as specified in:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant
//
// When a client is authorized all API requests will
// have the access token attached (will be authorized).
func (a *Anilist) AuthorizeWithCodeGrant(ctx context.Context, codeGrant CodeGrant) error {
	a.logger.Log("authenticating Anilist via code grant")
	// Access token
	token, err := a.getTokenFromCode(ctx, codeGrant)
	if err != nil {
		return AuthError(err.Error())
	}
	a.token = token

	// Get authenticated user (this verifies the token)
	user, err := a.getAuthenticatedUser(ctx)
	if err != nil {
		// remove token as it's possible it's not valid
		a.token = ""
		return AuthError(err.Error())
	}
	a.user = user

	// Store the user and token to cache
	if err := a.store.setUser(user.Name, user); err != nil {
		return AuthError(err.Error())
	}
	if err := a.store.setAuthToken(user.Name, token); err != nil {
		return AuthError(err.Error())
	}
	return nil
}

// AuthorizeWithAccessToken will authorize the client (login)
// via a pre-obtained access token, as specified in:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/implicit-grant
//
// When a client is authorized all API requests will
// have the access token attached (will be authorized).
func (a *Anilist) AuthorizeWithAccessToken(ctx context.Context, token string) error {
	a.logger.Log("authenticating Anilist via acces token")
	a.token = token

	// Get authenticated user (this verifies the token)
	user, err := a.getAuthenticatedUser(ctx)
	if err != nil {
		// remove token as it's possible it's not valid
		a.token = ""
		return AuthError(err.Error())
	}
	a.user = user

	// Store the user and token to cache
	if err := a.store.setUser(user.Name, user); err != nil {
		return AuthError(err.Error())
	}
	if err := a.store.setAuthToken(user.Name, token); err != nil {
		return AuthError(err.Error())
	}
	return nil
}

func (a *Anilist) getTokenFromCode(ctx context.Context, codeGrant CodeGrant) (string, error) {
	if err := codeGrant.Validate(); err != nil {
		return "", err
	}
	body, err := json.Marshal(codeGrant.reqBody())
	if err != nil {
		return "", err
	}

	resp, err := a.genericRequest(ctx, http.MethodPost, oAuthTokenURL, bytes.NewBuffer(body), false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("non-OK status response code: " + resp.Status)
	}

	var res oAuthData
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return "", err
	}
	return res.AccessToken, nil
}

// getAuthenticatedUser will query for the user data to the Anilist API.
func (a *Anilist) getAuthenticatedUser(ctx context.Context) (User, error) {
	body := apiRequestBody{
		Query: queryViewer,
	}
	data, err := sendRequest[userData](ctx, a, body)
	if err != nil {
		return User{}, errors.New("getting authenticated user data: " + err.Error())
	}
	return data.Viewer, nil
}

// Authenticated returns true if there is a currently authenticated
// user (there exists an available access token and user data).
func (a *Anilist) Authenticated() bool {
	return a.token != ""
}
