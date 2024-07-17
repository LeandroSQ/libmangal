package anilist

import (
	"net/http"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

// Options is options for Anilist client
type Options struct {
	// HTTPClient is a http client used for Anilist API
	HTTPClient *http.Client

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// LogWriter used for logs progress
	Logger *logger.Logger
}

// DefaultOptions constructs default AnilistOptions
func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{},
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
		Logger: logger.NewLogger(),
	}
}
