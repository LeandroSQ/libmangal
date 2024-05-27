package libmangal

import "fmt"

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

	// AnilistManga returns the set AnilistManga.
	// This is used to fetch metadata when downloading chapters.
	//
	// Also Useful for templates.
	AnilistManga() (AnilistManga, error)

	// SetAnilistManga will provide an AnilistManga for internal use.
	//
	// This is controlled on the client, not by libmangal.
	SetAnilistManga(AnilistManga)
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
	SeriesJSON() (seriesJSON SeriesJSON, found bool, err error)
}

// VolumeInfo is the general information for the volume.
type VolumeInfo struct {
	// Number of the volume.
	Number float32 `json:"number"`
}

// Volume of a manga. If a series is popular enough, its chapters
// are then collected and published into volumes,
// which usually feature a few chapters of the overall story.
// Most Manga series are long-running and can span multiple volumes.
//
// At least one volume is expected.
type Volume interface {
	fmt.Stringer

	Info() VolumeInfo

	// Manga gets the Manga that this Volume is relevant to.
	//
	// Implementation should not make any external requests
	// nor be computationally heavy.
	Manga() Manga
}

// ChapterInfo is the general information for the chapter.
type ChapterInfo struct {
	// Title is the title of chapter.
	Title string `json:"title"`

	// URL is the url leading to chapter web page.
	URL string `json:"url"`

	// Number of the chapter.
	//
	// Float type allows for extra chapters that usually have
	// numbering like the following: 10.5, 101.1, etc..
	Number float32 `json:"number"`

	// Date is the chapter publication date.
	Date Date `json:"date"`

	// ScanlationGroup is the group that did the scan/translation.
	//
	// If not an official publication, most of the chapters will belong
	// to a scanlation group.
	ScanlationGroup string `json:"scanlation_group"`
}

// Chapter is what Volume consists of. Each chapter is about 24–40 pages.
type Chapter interface {
	fmt.Stringer

	Info() ChapterInfo

	// Volume gets the Volume that this Chapter is relevant to.
	//
	// Implementation should not make any external requests
	// nor be computationally heavy.
	Volume() Volume
}

// ChapterWithComicInfoXML is a Chapter with an already associated ComicInfoXML.
//
// The associated ComicInfoXML will be used instead of the one generated from the metadata.
type ChapterWithComicInfoXML interface {
	Chapter

	// ComicInfoXML will be used to write ComicInfo.xml file.
	//
	// Implementation should not make any external requests.
	// If found is false then mangal will try to search on Anilist for the
	// relevant manga.
	ComicInfoXML() (comicInfoXML ComicInfoXML, found bool, err error)
}

// Page is what Chapter consists of.
type Page interface {
	fmt.Stringer

	// Extension gets the image extension of this page.
	// An extension must start with a dot.
	//
	// For example: .jpeg .png
	Extension() string

	// Chapter gets the Chapter that this Page is relevant to.
	//
	// Implementation should not make any external requests
	// nor be computationally heavy.
	Chapter() Chapter
}

// PageWithImage is a Page with already downloaded image.
//
// The associated image will be used instead of downloading one.
type PageWithImage interface {
	Page

	// Image gets the image contents.
	//
	// Implementation should not make any external requests.
	// Should only be exposed if the Page already contains image contents.
	Image() []byte

	// SetImage sets the image contents. This is used by DownloadOptions.ImageTransformer.
	SetImage(newImage []byte)
}

type pageWithImage struct {
	Page
	image []byte
}

func (p *pageWithImage) Image() []byte {
	return p.image
}

func (p *pageWithImage) SetImage(newImage []byte) {
	p.image = newImage
}
