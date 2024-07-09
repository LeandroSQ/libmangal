package mangadata

import (
	"fmt"

	"github.com/luevano/libmangal/metadata"
)

// MangaInfo is the general indispensable information for the manga.
type MangaInfo struct {
	// Title of the manga.
	Title string `json:"title"`

	// AnilistSearch is the title that will be used for
	// an Anilist search.
	//
	// This is a separate field as Title could be on any
	// language, but Anilist only supports searching for
	// english, native and romaji titles.
	AnilistSearch string `json:"anilist_search"`

	// URL leading to manga page web page.
	URL string `json:"url"`

	// ID of the Manga.
	//
	// It must be unique within its provider. It will be
	// part of the URL in most cases.
	ID string `json:"id"`

	// Cover is the cover image url.
	Cover string `json:"cover"`

	// Banner is the banner image url.
	//
	// Not all providers contain a banner image.
	Banner string `json:"banner"`
}

// Manga should provide basic information and its metadata found in the provider.
type Manga interface {
	fmt.Stringer

	Info() MangaInfo

	// Metadata gets the associated metadata of the manga.
	//
	// In its unchanged state, it's the basic metadata that is found in the provider itself.
	Metadata() *metadata.Metadata

	// SetMetadata will replace the current metadata.
	//
	// Useful when updating metadata fields. Its implementation should keep the
	// same pointer address intact, only updating the underlying data.
	SetMetadata(metadata *metadata.Metadata)
}

// MangaWithSeriesJSON is a Manga with an already associated SeriesJSON.
//
// The associated SeriesJSON will be used instead of the one generated from the metadata.
type MangaWithSeriesJSON interface {
	Manga

	// SeriesJSON will be used to write series.json file.
	//
	// Implementation should not make any external requests.
	// If found is false then mangal will try to search on Anilist for the
	// relevant manga.
	SeriesJSON() (seriesJSON metadata.SeriesJSON, found bool, err error)
}
