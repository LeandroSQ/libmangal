package myanimelist

import (
	"net/http"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

type Options struct {
	// ClientID of the MyAnimeList API client. Required.
	ClientID string

	// NSFW if NSFW mangas should be included in the searches.
	NSFW bool

	// HTTPClient is a http client used for Anilist API.
	HTTPClient *http.Client

	// CacheStore returns a gokv.Store implementation for use as a cache storage.
	CacheStore func(dbName, bucketName string) (gokv.Store, error)

	// LogWriter used for logs progress.
	Logger *logger.Logger
}

// DefaultOptions constructs default AnilistOptions.
//
// Note: the ClientID still needs to be passed separately.
func DefaultOptions() Options {
	return Options{
		NSFW:       false,
		HTTPClient: &http.Client{},
		CacheStore: func(dbName, bucketName string) (gokv.Store, error) {
			return syncmap.NewStore(syncmap.DefaultOptions), nil
		},
		Logger: logger.NewLogger(),
	}
}
