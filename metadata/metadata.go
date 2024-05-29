package metadata

import "fmt"

const (
	FilenameComicInfoXML = "ComicInfo.xml"
	FilenameSeriesJSON   = "series.json"
	FilenameCoverJPG     = "cover.jpg"
	FilenameBannerJPG    = "banner.jpg"
)

type Status string

const (
	StatusFinished       Status = "FINISHED"
	StatusReleasing      Status = "RELEASING"
	StatusNotYetReleased Status = "NOT_YET_RELEASED"
	StatusCancelled      Status = "CANCELLED"
	StatusHiatus         Status = "HIATUS"
)

// Date is a simple date holder.
type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

func (d Date) String() string {
	return fmt.Sprintf("%d-%02d-%02d", d.Year, d.Month, d.Day)
}

// Metadata is the general metadata information about a manga.
//
// In its most basic form, it's just the metadata that is available from the provider.
// It contains the necessary fields to build the series.json and ComicInfo.xml files.
type Metadata struct {
	// English is the english title of the manga.
	EnglishTitle string `json:"english_title"`

	// Romaji is the romanized title of the manga.
	RomajiTitle string `json:"romaji_title"`

	// Native is the native title of the manga. (Usually in kanji)
	NativeTitle string `json:"native_title"`

	// AlternateTitles that are known, in order of relevance.
	AlternateTitles []string `json:"alternate_titles"`

	// Score is the community score for the manga.
	//
	// Accepted values are between 0.0 and 5.0.
	Score float32 `json:"score"`

	// Description is the description/summary for the manga.
	Description string `json:"description"`

	// CoverImage is the cover image of the manga.
	CoverImage string `json:"cover_image"`

	// BannerImage is the banner image of the manga.
	BannerImage string `json:"banner_image"`

	// Tags is the list of tags associated with the manga.
	Tags []string `json:"tags"`

	// Genres is the list of genres associated with the manga.
	Genres []string `json:"genres"`

	// Characters is the list of characters, in order of relevance.
	Characters []string `json:"characters"`

	// Authors (or Writers) is the list of authors, in order of relevance.
	// Must contain at least one artist.
	Authors []string `json:"authors"`

	// Artists is the list of artists, in order of relevance.
	Artists []string `json:"artists"`

	// Translators is the list of translators, in order of relevance.
	Translators []string `json:"translators"`

	// Letterers is the list of letterers, in order of relevance.
	Letterers []string `json:"letterers"`

	// StartDate is the date the manga started publishing.
	StartDate Date `json:"start_date"`

	// EndDate is the date the manga ended publishing.
	EndDate Date `json:"end_date"`

	// Publisher of the manga.
	Publisher string `json:"publisher"`

	// Current status of the manga.
	//
	// One of: FINISHED, RELEASING, NOT_YET_RELEASED, CANCELLED, HIATUS
	Status Status `json:"status"`

	// Format the original publication.
	//
	// For example: TBP, HC, Web, Digital, etc..
	Format string `json:"format"`

	// Country of origin of the manga. ISO 3166-1 alpha-2 country code.
	Country string `json:"country"`

	// Chapter count until this point.
	Chapters int `json:"chapters"`

	// Extra notes to be added.
	Notes string `json:"notes"`

	// URL is the source URL of the metadata.
	URL string `json:"url"`

	// IDAl is the Anilist ID.
	IDAl int `json:"id_al"`

	// IDMal is the MyAnimeList ID.
	IDMal int `json:"id_mal"`
}

// String representation should be something short and informative.
//
// Returns representation in the style of "Title (Year) [IDType-ID]",
// if Year or ID is missing, then they're omitted.
func (m *Metadata) String() string {
	title := m.Title()
	year := m.StartDate.Year
	idSource := m.IDSource()
	id := m.ID()

	var yearStr string
	if year != 0 {
		yearStr = fmt.Sprintf(" (%d)", year)
	}
	var idStr string
	if id != 0 {
		idStr = fmt.Sprintf(" [%sid-%d]", idSource, id)
	}

	return fmt.Sprintf("%s%s%s", title, yearStr, idStr)
}

func (m *Metadata) Title() string {
	if m.EnglishTitle != "" {
		return m.EnglishTitle
	}
	if m.RomajiTitle != "" {
		return m.RomajiTitle
	}
	return m.NativeTitle
}

// ID of the external metadata.
func (m *Metadata) ID() int {
	// The anilist id should never be 0 (for now) but just in case
	if m.IDAl != 0 {
		return m.IDAl
	}
	return m.IDMal
}

// TODO: placeholder, need to better handle this
// once the amount of external metadata ids grow
//
// IDSource get the type of the external metadata ID.
//
// For example "al" for Anilist or "mal" for MyAnimeList.
func (m *Metadata) IDSource() string {
	if m.IDAl != 0 {
		return "al"
	}
	return "mal"
}
