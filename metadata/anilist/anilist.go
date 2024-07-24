package anilist

import (
	"context"
	"errors"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/metadata"
	"github.com/philippgille/gokv"
)

const apiURL = "https://graphql.anilist.co"

var info = metadata.ProviderInfo{
	ID:      metadata.IDCodeAnilist,
	Source:  metadata.IDSourceAnilist,
	Name:    "Anilist",
	Version: "0.1.0",
	Website: "https://anilist.co/",
}

var _ metadata.Provider = (*Anilist)(nil)

// Anilist is the Anilist client.
type Anilist struct {
	// authenticated user info
	user  metadata.User
	token string

	options Options
	logger  *logger.Logger
	store   store // still currently needed for the auth/user
}

// NewAnilist constructs new Anilist client.
func NewAnilist(options Options) (*Anilist, error) {
	s := store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return options.CacheStore(string(info.ID), bucketName)
		},
	}

	l := options.Logger
	if l == nil {
		l = logger.NewLogger()
	}
	anilist := &Anilist{
		options: options,
		logger:  l,
		store:   s,
	}

	// If no username provided, don't even check for token/user data
	if options.Username == "" {
		return anilist, nil
	}

	found, err := anilist.AuthorizeCachedUser(options.Username)
	if err != nil {
		return nil, err
	}
	if !found {
		anilist.logger.Log("no cached authentication data found for user %q", options.Username)
	}

	return anilist, nil
}

func (p *Anilist) String() string {
	return info.Name
}

// Info information about Provider.
func (p *Anilist) Info() metadata.ProviderInfo {
	return info
}

// SetLogger sets logger to use for this provider.
//
// Setting a nil logger will create a new one.
func (p *Anilist) SetLogger(_logger *logger.Logger) {
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
func (p *Anilist) Logger() *logger.Logger {
	return p.logger
}

// SearchByID for metadata with the given id.
//
// Implementation should only handle the request and and marshaling.
func (p *Anilist) SearchByID(ctx context.Context, id int) (metadata.Metadata, bool, error) {
	p.logger.Log("searching manga with id %d on Anilist", id)

	body := apiRequestBody{
		Query: querySearchByID,
		Variables: map[string]any{
			"id": id,
		},
	}
	data, err := sendRequest[byIDData](ctx, p, body)
	if err != nil {
		return nil, false, err
	}

	manga := data.Media
	if manga == nil {
		return nil, false, nil
	}

	return manga, true, nil
}

// Search for metadata with the given query.
//
// Implementation should only handle the request and and marshaling.
func (p *Anilist) Search(ctx context.Context, query string) ([]metadata.Metadata, error) {
	body := apiRequestBody{
		Query: querySearchByName,
		Variables: map[string]any{
			"query": query,
		},
	}
	data, err := sendRequest[mangasData](ctx, p, body)
	if err != nil {
		return nil, err
	}

	mangas := data.Page.Media.GetAsMetas()
	p.logger.Log("found %d manga(s) on Anilist", len(mangas))
	return mangas, nil
}

// SetMangaProgress sets the reading progress for a given manga metadata id.
func (p *Anilist) SetMangaProgress(ctx context.Context, id, chapterNumber int) error {
	if id == 0 {
		return Error("Anilist ID not valid (0)")
	}
	if !p.Authenticated() {
		return Error("not authorized")
	}

	body := apiRequestBody{
		Query: mutationSaveProgress,
		Variables: map[string]any{
			"id":       id,
			"progress": chapterNumber,
		},
	}
	_, err := sendRequest[setProgressData](ctx, p, body)
	if err != nil {
		return Error(err.Error())
	}

	return nil
}

// User returns the currently authenticated user.
func (p *Anilist) User() (metadata.User, error) {
	if !p.Authenticated() {
		return nil, errors.New("Anilist is not authenticated")
	}
	return p.user, nil
}
