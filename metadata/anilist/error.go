package anilist

type Error string

func (e Error) Error() string {
	return "anilist: " + string(e)
}

// OAuthError is used for generic Anilist OAuth errors.
type OAuthError string

func (e OAuthError) Error() string {
	return "anilist oauth: " + string(e)
}

type OAuthConvertCodeError string

func (e OAuthConvertCodeError) Error() string {
	return "anilist oauth converting access code to access token: " + string(e)
}

// Some specific errors

var (
	OAuthErrEmptyClientID    = OAuthError("ClientID is empty")
	OAuthErrEmptyCode        = OAuthError("Access Code/Token is empty")
	OAuthErrNilToken         = OAuthError("Token is nil")
	OAuthErrTooManyCallbacks = OAuthError("Too many callback requests")
)
