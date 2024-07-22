package myanimelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var mangaFields = strings.Join([]string{
	"id",
	"title",
	"main_picture",
	"alternative_titles",
	"start_date",
	"end_date",
	"synopsis",
	"mean",
	"rank",
	"popularity",
	"nsfw",
	"genres",
	"created_at",
	"updated_at",
	"media_type",
	"status",
	"num_volumes",
	"num_chapters",
	"authors{first_name,last_name}",
}, ",")

type mangasResponse struct {
	Data   mangaNodes `json:"data"`
	Paging struct {
		Previous string `json:"previous"`
		Next     string `json:"next"`
	} `json:"paging"`
}

type mangaNodes []mangaNode

func (n mangaNodes) Get() []Manga {
	mangas := make([]Manga, len(n))
	for i, n := range n {
		mangas[i] = n.Node
	}
	return mangas
}

type mangaNode struct {
	Node Manga `json:"node"`
}

func (a *MyAnimeList) request(
	ctx context.Context,
	path string,
	params url.Values,
	res any,
) error {
	if a.options.NSFW {
		params.Set("nsfw", "true")
	} else {
		params.Set("nsfw", "false")
	}

	params.Set("fields", mangaFields)
	u, _ := url.Parse(apiURL)
	u = u.JoinPath(path)
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// TODO: allow user login
	req.Header.Set("X-MAL-CLIENT-ID", a.options.ClientID)

	resp, err := a.options.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}
