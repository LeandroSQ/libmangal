package anilist

import (
	"net/http"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

// Options is options for Anilist client.
type Options struct {
	// Username is the authentication username.
	//
	// It's only used to check if it has been authenticated
	// previously and get its stored data and access token if available.
	//
	// An empty Username results in unauthenticated
	// Anilist client and doesn't error out.
	Username string

	// HTTPClient is a http client used for Anilist API.
	HTTPClient *http.Client

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// LogWriter used for logs progress.
	Logger *logger.Logger
}

// DefaultOptions constructs default AnilistOptions.
func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
		Logger: logger.NewLogger(),
	}
}
