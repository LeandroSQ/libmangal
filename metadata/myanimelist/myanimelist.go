package myanimelist

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
)

const (
	apiURL      = "https://api.myanimelist.net/v2"
	CacheDBName = "myanimelist"
)

// MyAnimeList is the MyAnimeList client.
type MyAnimeList struct {
	options Options
	logger  *logger.Logger
	store   store
}

// NewMAL constructs new MyAnimeList client.
func NewMAL(options Options) (*MyAnimeList, error) {
	if options.ClientID == "" {
		return nil, errors.New("MAL ClientID must not be empty")
	}

	s := store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return options.CacheStore(CacheDBName, bucketName)
		},
	}

	mal := &MyAnimeList{
		options: options,
		logger:  options.Logger,
		store:   s,
	}

	return mal, nil
}

// SearchByID gets mal manga by its id.
func (a *MyAnimeList) SearchByID(ctx context.Context, id int) (Manga, bool, error) {
	a.logger.Log("searching manga with id %d on MyAnimeList", id)
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

func (a *MyAnimeList) searchByID(ctx context.Context, id int) (Manga, bool, error) {
	params := url.Values{}
	params.Set("manga_id", strconv.Itoa(id))

	var manga *Manga
	err := a.request(ctx, "manga/"+strconv.Itoa(id), params, &manga)
	if err != nil {
		return Manga{}, false, err
	}

	if manga == nil {
		return Manga{}, false, nil
	}

	return *manga, true, nil
}

// SearchMangas gets a list of mal mangas given a query (title).
func (a *MyAnimeList) SearchMangas(ctx context.Context, query string) ([]Manga, error) {
	a.logger.Log("searching mangas with query %q on MyAnimeList", query)
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

func (a *MyAnimeList) searchMangas(ctx context.Context, query string) ([]Manga, error) {
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
