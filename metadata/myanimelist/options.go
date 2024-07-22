package myanimelist

import (
	"net/http"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
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
