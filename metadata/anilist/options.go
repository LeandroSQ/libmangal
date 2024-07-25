package anilist

import (
	"net/http"

	"github.com/luevano/libmangal/logger"
)

// Options is options for Anilist client.
type Options struct {
	// HTTPClient is a http client used for Anilist API.
	HTTPClient *http.Client

	// LogWriter used for logs progress.
	Logger *logger.Logger
}

// DefaultOptions constructs default AnilistOptions.
func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		Logger:     logger.NewLogger(),
	}
}
