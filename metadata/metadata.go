package metadata

import (
	"fmt"
	"reflect"
)

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

const (
	IDSourceAnilist     = "al"
	IDSourceMyAnimeList = "mal"
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

	// IDProvider is the provider assigned ID
	// (used when no other metadata ID is available),
	// could be arbitrary.
	//
	// If set, IDProviderName must also be set and it
	// is considered that this metadata is from the provider.
	IDProvider int `json:"id_provider"`

	// IDProviderName is the provider ID name,
	// goes alongside IDProvider, could be arbitrary.
	//
	// Should be less than 5 characters, ideally just 2-3.
	//
	// If set, it is considered that this metadata is from the provider.
	// Regardless of IDProvider value.
	IDProviderName string `json:"id_provider_name"`

	// IDAl is the Anilist ID.
	//
	// Must be > 0.
	IDAl int `json:"id_al"`

	// IDMal is the MyAnimeList ID.
	//
	// Must be > 0.
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
	if id > 0 && idSource != "" {
		idStr = fmt.Sprintf(" [%sid-%d]", idSource, id)
	}

	return fmt.Sprintf("%s%s%s", title, yearStr, idStr)
}

// Title returns the title of the manga.
//
// Order of priority:
// English -> Romaji -> Native
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
//
// Checks for an ID in the following order:
// Anilist -> MyAnimeList -> Provider (custom)
//
// If no ID is available, returns -1. If IDProviderName is non-empty,
// it returns the IDProvider regardless of its value, to take into account provider-only metadata.
func (m *Metadata) ID() int {
	// Sometimes there is no provider metadata ID (mangadex), so check for the name.
	if m.IDProviderName != "" {
		return m.IDProvider
	}
	// The anilist id should never be 0 (for now) but just in case.
	if m.IDAl > 0 {
		return m.IDAl
	}
	if m.IDMal > 0 {
		return m.IDMal
	}
	return -1
}

// IDSource get the type of the external metadata ID.
//
// For example "al" for Anilist or "mal" for MyAnimeList.
//
// Checks for an ID in the following order:
// Anilist -> MyAnimeList -> Provider (custom)
//
// If no ID available, returns empty string. If IDProviderName is non-empty,
// it returns that as the first option, to take into account provider-only metadata.
func (m *Metadata) IDSource() string {
	// Same as ID() logic.
	if m.IDProviderName != "" {
		return m.IDProviderName
	}
	if m.IDAl > 0 {
		return IDSourceAnilist
	}
	if m.IDMal > 0 {
		return IDSourceMyAnimeList
	}
	return ""
}

// Validate will make sure the Metadata is valid/usable
// to write enough metadata to files.
//
// At the very least checks that: Title, Description, Authors,
// StartDate and Status are non-empty/non-zero.
func (m *Metadata) Validate() error {
	if m == nil {
		return Error{fmt.Errorf("Metadata is nil")}
	}

	if m.Title() == "" {
		return Error{fmt.Errorf("Title must be non-empty")}
	}
	// Some descriptions are empty even on actual metadata providers
	// if m.Description == "" {
	// 	return Error{fmt.Errorf("Description must be non-empty")}
	// }
	if len(m.Authors) == 0 {
		return Error{fmt.Errorf("Authors must be non-empty")}
	}
	if m.StartDate == (Date{}) {
		return Error{fmt.Errorf("StartDate must be non-zero")}
	}
	if m.Status == "" {
		return Error{fmt.Errorf("Status must be non-empty")}
	}
	// TODO: also check for ID/IDSource? Metadata.String allows for missing id...
	//
	// if m.ID() == -1 {
	// 	return Error{fmt.Errorf("ID must be greater than zero")}
	// }
	// if m.IDSource() == "" {
	// 	return Error{fmt.Errorf("IDSource must be non-empty")}
	// }
	return nil
}

// ValidityScore returns a value that represents the amount
// of Metadata fields set.
//
// 0 means it's the zero-valued Metadata, -1 that it's null.
func (m *Metadata) ValidityScore() int {
	if m == nil {
		return -1
	}

	s := 0
	mValue := reflect.ValueOf(*m)
	for i := 0; i < mValue.NumField(); i++ {
		// fieldName := mValue.Type().Field(i).Name
		fieldValue := mValue.Field(i).Interface()
		if !reflect.DeepEqual(fieldValue, reflect.Zero(mValue.Field(i).Type()).Interface()) {
			s += 1
		}
	}
	return s
}
