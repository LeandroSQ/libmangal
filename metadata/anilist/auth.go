package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// cacheAccessTokenKey is the key used to store Anilist access code.
// It's needed, since the KV interface always expects a key to be passed.
const (
	cacheAccessTokenKey = "hi"
	oAuthPinURL         = "https://anilist.co/api/v2/oauth/pin"
	oAuthTokenURL       = "https://anilist.co/api/v2/oauth/token"
)

type LoginCredentials struct {
	ID     string
	Secret string
	Code   string
}

func (c LoginCredentials) Validate() error {
	if c.ID == "" {
		return LoginCredentialsError("ID is empty")
	}
	if c.Secret == "" {
		return LoginCredentialsError("Secret is empty")
	}
	if c.Code == "" {
		return LoginCredentialsError("Code is empty")
	}
	return nil
}

func (c LoginCredentials) reqBody() map[string]string {
	return map[string]string{
		"client_id":     c.ID,
		"client_secret": c.Secret,
		"code":          c.Code,
		"grant_type":    "authorization_code",
		"redirect_uri":  oAuthPinURL,
	}
}

type authResponse struct {
	AccessToken string `json:"access_token"`
}

func (a *Anilist) Logout() error {
	return a.store.deleteAuthToken(cacheAccessTokenKey)
}

// Authorize will obtain Anilist token for API requests.
func (a *Anilist) Authorize(
	ctx context.Context,
	credentials LoginCredentials,
) error {
	a.logger.Log("logging into Anilist")
	if err := credentials.Validate(); err != nil {
		return err
	}

	body, err := json.Marshal(credentials.reqBody())
	if err != nil {
		return Error(err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oAuthTokenURL, bytes.NewBuffer(body))
	if err != nil {
		return Error(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.options.HTTPClient.Do(req)
	if err != nil {
		return Error(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Error("non-OK status response code: " + resp.Status)
	}

	var res authResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return Error(err.Error())
	}

	if err := a.store.setAuthToken(cacheAccessTokenKey, res.AccessToken); err != nil {
		return err
	}

	a.accessToken = res.AccessToken
	return nil
}

func (a *Anilist) IsAuthorized() bool {
	return a.accessToken != ""
}
