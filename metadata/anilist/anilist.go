package anilist

import (
	"context"
	"strings"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/philippgille/gokv"
)

const (
	apiURL      = "https://graphql.anilist.co"
	CacheDBName = "anilist"
)

// Anilist is the Anilist client.
type Anilist struct {
	accessToken string
	options     Options
	logger      *logger.Logger
	store       store
}

// NewAnilist constructs new Anilist client.
func NewAnilist(options Options) (*Anilist, error) {
	s := store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return options.CacheStore(CacheDBName, bucketName)
		},
	}

	anilist := &Anilist{
		options: options,
		logger:  options.Logger,
		store:   s,
	}
	accessToken, found, err := s.getAuthToken(cacheAccessTokenKey)
	if err != nil {
		return nil, err
	}
	if found {
		anilist.accessToken = accessToken
	}
	return anilist, nil
}

// SearchByID gets anilist manga by its id.
func (a *Anilist) SearchByID(
	ctx context.Context,
	id int,
) (Manga, bool, error) {
	manga, found, err := a.store.getManga(id)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}
	if found {
		return manga, true, nil
	}

	manga, ok, err := a.searchByID(ctx, id)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}
	if !ok {
		return Manga{}, false, nil
	}

	err = a.store.setManga(id, manga)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}

	return manga, true, nil
}

func (a *Anilist) searchByID(
	ctx context.Context,
	id int,
) (Manga, bool, error) {
	a.logger.Log("searching manga with id %d on Anilist", id)

	body := apiRequestBody{
		Query: querySearchByIDm,
		Variables: map[string]any{
			"id": id,
		},
	}
	data, err := sendRequest[byIDData](ctx, a, body)
	if err != nil {
		return Manga{}, false, err
	}

	manga := data.Media
	if manga == nil {
		return Manga{}, false, nil
	}

	return *manga, true, nil
}

// TODO: re-validate that SearchMangas is working as intended.
//
// SearchMangas gets a list of anilist mangas given a query (title).
func (a *Anilist) SearchMangas(
	ctx context.Context,
	query string,
) ([]Manga, error) {
	a.logger.Log("searching mangas with query %q on Anilist", query)

	ids, found, err := a.store.getQueryIDs(query)
	if err != nil {
		return nil, Error(err.Error())
	}
	if found {
		var mangas []Manga
		for _, id := range ids {
			manga, ok, err := a.SearchByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if ok {
				mangas = append(mangas, manga)
			}
		}
		return mangas, nil
	}

	mangas, err := a.searchMangas(ctx, query)
	if err != nil {
		return nil, err
	}

	ids = make([]int, len(mangas))
	for i, manga := range mangas {
		err := a.store.setManga(manga.IDProvider, manga)
		if err != nil {
			return nil, Error(err.Error())
		}

		ids[i] = manga.IDProvider
	}

	err = a.store.setQueryIDs(query, ids)
	if err != nil {
		return nil, Error(err.Error())
	}

	return mangas, nil
}

func (a *Anilist) searchMangas(
	ctx context.Context,
	query string,
) ([]Manga, error) {
	body := apiRequestBody{
		Query: querySearchByName,
		Variables: map[string]any{
			"query": query,
		},
	}
	data, err := sendRequest[mangasData](ctx, a, body)
	if err != nil {
		return nil, err
	}

	mangas := data.Page.Media
	a.logger.Log("found %d manga(s) on Anilist", len(mangas))

	return mangas, nil
}

// SearchByManga is a convenience method to search given a Manga.
//
// Tries to search anilist manga in the following order:
//
// 1. If the manga contains non-nil metadata, by its Anilist ID if available.
//
// 2. If the manga title (priority on AnilistSearch field, then Title field) is binded to an Anilist ID.
//
// 3. Otherwise find closest anilist manga (FindClosestManga) by using the manga Title (priority on AnilistSearch field) field.
func (a *Anilist) SearchByManga(
	ctx context.Context,
	manga mangadata.Manga,
) (Manga, bool, error) {
	a.logger.Log("finding manga by (libmangal) manga on Anilist")
	meta := manga.Metadata()

	// Try to search by Anilist ID if it is available
	for _, id := range meta.ExtraIDs() {
		if id.Code == metadata.IDCodeAnilist {
			anilistManga, found, err := a.SearchByID(ctx, id.Value())
			if err == nil && found {
				return anilistManga, true, nil
			}
		}
	}

	// Else try to search by the title, this doesn't ensure
	// that the found anilist manga is 100% corresponding to
	// the manga requested, there are some instances in which
	// the result is wrong
	title := manga.Info().AnilistSearch
	if title == "" {
		title = manga.Info().Title
	}

	return a.FindClosestManga(ctx, title)
}

// FindClosestManga gets the manga with the title closest to the queried title.
func (a *Anilist) FindClosestManga(
	ctx context.Context,
	title string,
) (Manga, bool, error) {
	a.logger.Log("finding closest manga with query %q on Anilist", title)

	id, found, err := a.store.getTitleID(title)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}
	if found {
		manga, found, err := a.store.getManga(id)
		if err != nil {
			return Manga{}, false, Error(err.Error())
		}

		if found {
			return manga, true, nil
		}
	}

	manga, ok, err := a.findClosestManga(ctx, title, 3, 3)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}
	if !ok {
		return Manga{}, false, nil
	}

	err = a.store.setTitleID(title, manga.IDProvider)
	if err != nil {
		return Manga{}, false, Error(err.Error())
	}

	return manga, true, nil
}

func (a *Anilist) findClosestManga(
	ctx context.Context,
	title string,
	step,
	tries int,
) (Manga, bool, error) {
	for i := 0; i < tries; i++ {
		a.logger.Log("finding closest manga with query %q on Anilist (try %d/%d)", title, i+1, tries)

		mangas, err := a.SearchMangas(ctx, title)
		if err != nil {
			return Manga{}, false, err
		}

		if len(mangas) > 0 {
			closest := mangas[0]
			a.logger.Log("found closest manga on Anilist: %q with id %d", closest.String(), closest.ID)
			return closest, true, nil
		}

		// try again with a different title
		// remove `step` characters from the end of the title
		// avoid removing the last character or going out of bounds
		var newLen int

		title = strings.TrimSpace(title)

		if len(title) > step {
			newLen = len(title) - step
		} else if len(title) > 1 {
			newLen = len(title) - 1
		} else {
			break
		}

		title = title[:newLen]
	}

	return Manga{}, false, nil
}

// BindTitleWithID sets a given id to a title,
// so on each title search the same anilist manga with that id is obtained.
func (a *Anilist) BindTitleWithID(title string, id int) error {
	err := a.store.setTitleID(title, id)
	if err != nil {
		return Error(err.Error())
	}

	return nil
}

// SetMangaProgress sets the reading progress for a given anilist id.
func (a *Anilist) SetMangaProgress(ctx context.Context, id, chapterNumber int) error {
	if id == 0 {
		return Error("Anilist ID not valid (0)")
	}
	if !a.IsAuthorized() {
		return Error("not authorized")
	}

	body := apiRequestBody{
		Query: mutationSaveProgress,
		Variables: map[string]any{
			"id":       id,
			"progress": chapterNumber,
		},
	}
	_, err := sendRequest[setProgressData](ctx, a, body)
	if err != nil {
		return Error(err.Error())
	}

	return nil
}
