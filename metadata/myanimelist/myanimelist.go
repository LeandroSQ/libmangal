package myanimelist

import (
	"context"
	"errors"
	"net/url"

	"github.com/luevano/libmangal/logger"
)

const (
	apiURL      = "https://api.myanimelist.net/v2"
	CacheDBName = "myanimelist"
)

// MyAnimeList is the MyAnimeList client.
type MyAnimeList struct {
	options Options
	logger  *logger.Logger
}

// NewMAL constructs new MyAnimeList client.
func NewMAL(options Options) (*MyAnimeList, error) {
	if options.ClientID == "" {
		return nil, errors.New("MAL ClientID must not be empty")
	}
	mal := &MyAnimeList{
		options: options,
		logger:  options.Logger,
	}
	return mal, nil
}

// TODO: add caching
//
// SearchMangas gets a list of mal mangas given a query (title).
func (a *MyAnimeList) SearchMangas(ctx context.Context, query string) ([]Manga, error) {
	return a.searchMangas(ctx, query)
}

func (a *MyAnimeList) searchMangas(ctx context.Context, query string) ([]Manga, error) {
	a.logger.Log("searching mangas with query %q on MyAnimeList", query)
	params := url.Values{}
	params.Set("q", query)
	params.Set("offset", "0")
	params.Set("limit", "30")

	var res mangasResponse
	err := a.request(ctx, "manga", params, &res)
	if err != nil {
		return nil, err
	}

	mangas := res.Data.Get()
	a.logger.Log("found %d manga(s) on MyAnimeList", len(mangas))
	return mangas, nil
}
