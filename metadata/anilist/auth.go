package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// cacheAccessTokenKey is the key used to store Anilist access code.
// It's needed, since the KV interface always expects a key to be passed.
const (
	cacheAccessTokenKey = "hi"
	oAuthPinURL         = "https://anilist.co/api/v2/oauth/pin"
	oAuthTokenURL       = "https://anilist.co/api/v2/oauth/token"
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

func (a *Anilist) Logout() error {
	return a.store.deleteAuthToken(cacheAccessTokenKey)
}

// AuthorizeWithCodeGrant will authorize the client
// via code grant, as specified in:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant
//
// When a client is authorized all API requests will
// have the access token attached (will be authorized).
func (a *Anilist) AuthorizeWithCodeGrant(ctx context.Context, codeGrant CodeGrant) error {
	a.logger.Log("logging into Anilist via code grant")

	token, err := a.getTokenFromCode(ctx, codeGrant)
	if err != nil {
		return AuthError(err.Error())
	}

	if err := a.store.setAuthToken(cacheAccessTokenKey, token); err != nil {
		return AuthError(err.Error())
	}

	a.accessToken = token
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

func (a *Anilist) IsAuthorized() bool {
	return a.accessToken != ""
}
