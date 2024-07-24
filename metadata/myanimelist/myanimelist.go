package myanimelist

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/metadata"
)

const apiURL = "https://api.myanimelist.net/v2"

var info = metadata.ProviderInfo{
	ID:      metadata.IDCodeMyAnimeList,
	Source:  metadata.IDSourceMyAnimeList,
	Name:    "MyAnimeList",
	Version: "0.1.0",
	Website: "https://myanimelist.net/",
}

var _ metadata.Provider = (*MyAnimeList)(nil)

// MyAnimeList is a metadata.Provider implementation for MyAnimeList.
type MyAnimeList struct {
	options Options
	logger  *logger.Logger
}

// NewMAL constructs new MyAnimeList client.
func NewMAL(options Options) (*MyAnimeList, error) {
	if options.ClientID == "" {
		return nil, errors.New("MAL ClientID must not be empty")
	}

	l := options.Logger
	if l == nil {
		l = logger.NewLogger()
	}
	mal := &MyAnimeList{
		options: options,
		logger:  l,
	}

	return mal, nil
}

func (p *MyAnimeList) String() string {
	return info.Name
}

// Info information about Provider.
func (p *MyAnimeList) Info() metadata.ProviderInfo {
	return info
}

// SetLogger sets logger to use for this provider.
//
// Setting a nil logger will create a new one.
func (p *MyAnimeList) SetLogger(_logger *logger.Logger) {
	if _logger != nil {
		// p.logger is guaranteed to be non-nil
		*p.logger = *_logger
	} else {
		p.logger = logger.NewLogger()
	}
}

// Logger returns the set logger.
//
// Always returns a non-nil logger.
func (p *MyAnimeList) Logger() *logger.Logger {
	return p.logger
}

// SearchByID for metadata with the given id.
// Implementation should only handle the request and and marshaling.
func (p *MyAnimeList) SearchByID(ctx context.Context, id int) (metadata.Metadata, bool, error) {
	params := url.Values{}
	params.Set("manga_id", strconv.Itoa(id))

	var manga *Manga
	err := p.request(ctx, "manga/"+strconv.Itoa(id), params, &manga)
	if err != nil {
		return nil, false, err
	}

	if manga == nil {
		return nil, false, nil
	}

	return manga, true, nil
}

// Search for metadata with the given query.
//
// Implementation should only handle the request and and marshaling.
func (p *MyAnimeList) Search(ctx context.Context, query string) ([]metadata.Metadata, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("offset", "0")
	params.Set("limit", "30")

	var res mangasResponse
	err := p.request(ctx, "manga", params, &res)
	if err != nil {
		return nil, err
	}

	mangas := res.Data.GetAsMetas()
	p.logger.Log("found %d manga(s) on MyAnimeList", len(mangas))
	return mangas, nil
}

// FIX: implement
//
// SetMangaProgress sets the reading progress for a given manga metadata id.
func (p *MyAnimeList) SetMangaProgress(ctx context.Context, id, chapterNumber int) error {
	return errors.ErrUnsupported
}
