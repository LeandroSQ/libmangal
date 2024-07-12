package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

func ToSeriesJSON(m Metadata) SeriesJSON {
	var status string
	// TODO: need to better handle these statuses,
	// series.json only supports "ended" and "continuing"
	switch m.Status() {
	case StatusFinished:
		status = "Ended"
	case StatusReleasing:
		status = "Continuing"
	default:
		status = "Unknown"
	}

	// Format should be:
	// "November 2011 - July 2016"
	// "June 2021 - Present"
	var pubEndDate string
	if m.EndDate() != (Date{}) {
		pubEndDate = fmt.Sprintf(
			"%s %d",
			time.Month(m.EndDate().Month).String(),
			m.EndDate().Year)
	} else {
		pubEndDate = "Present"
	}

	publicationRun := fmt.Sprintf(
		"%s %d - %s",
		time.Month(m.StartDate().Month).String(),
		m.StartDate().Year,
		pubEndDate,
	)

	// TODO: clean the description at least for one of the fields?
	return SeriesJSON{
		Type:                 "comicSeries",
		Name:                 m.Title(),
		DescriptionFormatted: m.Description(),
		DescriptionText:      m.Description(),
		Status:               status,
		Year:                 m.StartDate().Year,
		ComicImage:           m.Cover(),
		Publisher:            m.Publisher(),
		ComicID:              m.ID().Value(),
		BookType:             "Print",
		TotalIssues:          m.Chapters(),
		PublicationRun:       publicationRun,
	}
}

// SeriesJSON v1.0.2 is similar to ComicInfoXML but designed for
// the series as a whole rather than a single chapter. Defined by MyLar.
//
// https://github.com/mylar3/mylar3/wiki/series.json-schema-%28version-1.0.2%29
type SeriesJSON struct {
	Type                 string `json:"type"`
	Name                 string `json:"name"`
	DescriptionFormatted string `json:"description_formatted"`
	DescriptionText      string `json:"description_text"`
	Status               string `json:"status"`
	Year                 int    `json:"year"`
	ComicImage           string `json:"comic_image"`
	Publisher            string `json:"publisher"`
	ComicID              int    `json:"comicid"`
	BookType             string `json:"booktype"`
	TotalIssues          int    `json:"total_issues"`
	PublicationRun       string `json:"publication_run"`
}

// TODO: need to decide if HTML escaping should be disabled
func (s SeriesJSON) Marshal() ([]byte, error) {
	// return json.MarshalIndent(s, "", "  ")
	buffer := &bytes.Buffer{}
	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "  ")
	err := enc.Encode(s.wrapper())

	return buffer.Bytes(), err
}

func (s SeriesJSON) wrapper() seriesJSONWrapper {
	return seriesJSONWrapper{Metadata: s}
}

type seriesJSONWrapper struct {
	Metadata SeriesJSON `json:"metadata"`
}
