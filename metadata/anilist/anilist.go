package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
)

const anilistAPIURL = "https://graphql.anilist.co"

type anilistRequestBody struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// Anilist is the Anilist client.
type Anilist struct {
	accessToken string
	options     Options
	logger      *logger.Logger
}

// NewAnilist constructs new Anilist client.
func NewAnilist(options Options) Anilist {
	var accessToken string
	found, err := options.AccessTokenStore.Get(anilistStoreAccessCodeStoreKey, &accessToken)

	anilist := Anilist{
		options: options,
		logger:  options.Logger,
	}

	if err == nil && found {
		anilist.accessToken = accessToken
	}

	return anilist
}

// SearchByID gets anilist manga by its id.
func (a *Anilist) SearchByID(
	ctx context.Context,
	id int,
) (Manga, bool, error) {
	manga, found, err := a.cacheStatusId(id)
	if err != nil {
		return Manga{}, false, Error{err}
	}
	if found {
		return manga, true, nil
	}

	manga, ok, err := a.searchByID(ctx, id)
	if err != nil {
		return Manga{}, false, Error{err}
	}
	if !ok {
		return Manga{}, false, nil
	}

	err = a.cacheSetId(id, manga)
	if err != nil {
		return Manga{}, false, Error{err}
	}

	return manga, true, nil
}

func (a *Anilist) searchByID(
	ctx context.Context,
	id int,
) (Manga, bool, error) {
	a.logger.Log("searching manga with id %d on Anilist", id)

	body := anilistRequestBody{
		Query: anilistQuerySearchByID,
		Variables: map[string]any{
			"id": id,
		},
	}

	data, err := sendRequest[struct {
		Media *Manga `json:"media"`
	}](ctx, a, body)
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

	ids, found, err := a.cacheStatusQuery(query)
	if err != nil {
		return nil, Error{err}
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
		err := a.cacheSetId(manga.ID, manga)
		if err != nil {
			return nil, Error{err}
		}

		ids[i] = manga.ID
	}

	err = a.cacheSetQuery(query, ids)
	if err != nil {
		return nil, Error{err}
	}

	return mangas, nil
}

func (a *Anilist) searchMangas(
	ctx context.Context,
	query string,
) ([]Manga, error) {
	body := anilistRequestBody{
		Query: anilistQuerySearchByName,
		Variables: map[string]any{
			"query": query,
		},
	}

	data, err := sendRequest[struct {
		Page struct {
			Media []Manga `json:"media"`
		} `json:"page"`
	}](ctx, a, body)
	if err != nil {
		return nil, err
	}

	mangas := data.Page.Media
	a.logger.Log("found %d manga(s) on Anilist", len(mangas))

	return mangas, nil
}

type anilistResponse[Data any] struct {
	Errors []struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"errors"`
	Data Data `json:"data"`
}

func sendRequest[Data any](
	ctx context.Context,
	anilist *Anilist,
	requestBody anilistRequestBody,
) (data Data, err error) {
	marshalled, err := json.Marshal(requestBody)
	if err != nil {
		return data, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistAPIURL, bytes.NewReader(marshalled))
	if err != nil {
		return data, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	if anilist.IsAuthorized() {
		request.Header.Set(
			"Authorization",
			fmt.Sprintf("Bearer %s", anilist.accessToken),
		)
	}

	response, err := anilist.options.HTTPClient.Do(request)
	if err != nil {
		return data, err
	}

	defer response.Body.Close()

	// https://anilist.gitbook.io/anilist-apiv2-docs/overview/rate-limiting
	if response.StatusCode == http.StatusTooManyRequests {
		retryAfter := response.Header.Get("X-RateLimit-Remaining")
		if retryAfter == "" {
			// 90 seconds
			retryAfter = "90"
		}

		seconds, err := strconv.Atoi(retryAfter)
		if err != nil {
			return data, err
		}

		anilist.logger.Log("rate limited, retrying in %d seconds", seconds)

		select {
		case <-time.After(time.Duration(seconds) * time.Second):
		case <-ctx.Done():
			return data, ctx.Err()
		}

		return sendRequest[Data](ctx, anilist, requestBody)
	}

	if response.StatusCode != http.StatusOK {
		return data, fmt.Errorf(response.Status)
	}

	var body anilistResponse[Data]

	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		return data, err
	}

	if body.Errors != nil {
		err := body.Errors[0]
		return data, errors.New(err.Message)
	}

	return body.Data, nil
}

// SearchByManga is a convenience method to search given a Manga.
//
// If manga contains non-nil metadata, it will try to search by its Anilist ID first if available, then by title.
func (a *Anilist) SearchByManga(
	ctx context.Context,
	manga mangadata.Manga,
) (Manga, bool, error) {
	a.logger.Log("finding manga by (libmangal) manga on Anilist")
	metadata := manga.Metadata()
	if metadata != nil && metadata.IDAl != 0 {
		anilistManga, found, err := a.SearchByID(ctx, metadata.IDAl)
		if err == nil && found {
			return anilistManga, true, nil
		}
	}

	var title string
	if manga.Info().AnilistSearch != "" {
		title = manga.Info().AnilistSearch
	} else {
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

	id, found, err := a.cacheStatusTitle(title)
	if err != nil {
		return Manga{}, false, Error{err}
	}
	if found {
		manga, found, err := a.cacheStatusId(id)
		if err != nil {
			return Manga{}, false, Error{err}
		}

		if found {
			return manga, true, nil
		}
	}

	manga, ok, err := a.findClosestManga(ctx, title, 3, 3)
	if err != nil {
		return Manga{}, false, Error{err}
	}
	if !ok {
		return Manga{}, false, nil
	}

	err = a.cacheSetTitle(title, manga.ID)
	if err != nil {
		return Manga{}, false, Error{err}
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

// BindTitleWithID sets a given id to a title, so on each title search the same manga with that id is obtained.
func (a *Anilist) BindTitleWithID(title string, mangaID int) error {
	err := a.options.TitleToIDStore.Set(title, mangaID)
	if err != nil {
		return Error{err}
	}

	return nil
}

// SetMangaProgress sets the reading progress for a given manga id.
func (a *Anilist) SetMangaProgress(ctx context.Context, mangaID, chapterNumber int) error {
	if !a.IsAuthorized() {
		return Error{errors.New("not authorized")}
	}

	_, err := sendRequest[struct {
		SaveMediaListEntry struct {
			ID int `json:"id"`
		} `json:"SaveMediaListEntry"`
	}](
		ctx,
		a,
		anilistRequestBody{
			Query: anilistMutationSaveProgress,
			Variables: map[string]any{
				"id":       mangaID,
				"progress": chapterNumber,
			},
		},
	)
	if err != nil {
		return Error{err}
	}

	return nil
}
