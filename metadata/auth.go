package metadata

// CodeGrant is the typical code grant type authentication.
//
// It is also possible to keep the Secret empty to use as
// an implicit grant. For example, when it's not possible to
// store the client secret securely (which will be most cases).
//
// For more:
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/getting-started#using-oauth,
// https://anilist.gitbook.io/anilist-apiv2-docs/overview/oauth/authorization-code-grant
// https://myanimelist.net/apiconfig/references/authorization#step-6-exchange-authorization-code-for-refresh-and-access-tokens
type CodeGrant struct {
	// ClientID is the application client ClientID.
	ClientID string `json:"client_id"`

	// ClientSecret is the application client secret.
	//
	// May be empty, in which case it's an implicit grant
	// for providers that support it.
	ClientSecret string `json:"client_secret"`

	// Code is the user provided access code. One time use.
	//
	// If Secret is empty, this is the access token,
	// for providers that support it.
	Code string `json:"code"`

	// RedirectURI callback to receive the access code.
	//
	// May be empty, in which case it uses the default
	// redirect URI for the provider.
	RedirectURI string `json:"redirect_uri"`
}

func (c CodeGrant) Validate() error {
	if c.ClientID == "" {
		return CodeGrantError("(Client)ID is empty")
	}
	if c.Code == "" {
		return CodeGrantError("Code/AccessToken is empty")
	}
	return nil
}

// ToReqBody is a convenience method to convert the CodeGrant
// into a map ready to be Marshaled. Useful to later on add
// more fields if needed.
func (c CodeGrant) ToReqBody() map[string]string {
	body := map[string]string{
		"client_id":  c.ClientID,
		"code":       c.Code,
		"grant_type": "authorization_code",
	}
	if c.ClientSecret != "" {
		body["client_secret"] = c.ClientSecret
	}
	if c.RedirectURI != "" {
		body["redirect_uri"] = c.RedirectURI
	}
	return body
}

// AuthData is the generalized access/refresh token
// data and their expiration (if any).
type AuthData struct {
	// AccessToken is the code used to make
	// Authorized requests.
	AccessToken string `json:"access_token"`

	// RefreshToken is used to refresh the
	// AccessToken after it's been expired.
	RefreshToken string `json:"refresh_token"`

	// CreatedAt is the Unix Timestamp at
	// which the AccessToken was created.
	CreatedAt int `json:"created_at"`

	// ExpiresIn is the time in seconds
	// before the AccessToken expires.
	ExpiresIn int `json:"expires_in"`

	// Scope is the scope of the access.
	//
	// May be empty.
	Scope string `json:"scope"`

	// TokenType is the type of the
	// AccessToken (usually "Bearer")
	TokenType string `json:"token_type"`
}
