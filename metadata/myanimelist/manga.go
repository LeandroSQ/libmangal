package myanimelist

// Manga is a metadata.Metadata implementation
// for MyAnimeList manga metadata.
//
// Note that Manga fields don't match the incoming json
// fields to avoid collisions with the interface.
type Manga struct {
	IDProvider    int    `json:"id"`
	TitleProvider string `json:"title"`
	MainPicture   struct {
		Large  string `json:"large"`
		Medium string `json:"medium"`
	} `json:"main_picture"`
	AlternativeTitles struct {
		Synonyms []string `json:"synonyms"`
		En       string   `json:"en"`
		Ja       string   `json:"ja"`
	} `json:"alternative_titles"`
	DateStart  string  `json:"start_date"`
	DateEnd    string  `json:"end_date"`
	Synopsis   string  `json:"synopsis"`
	Mean       float32 `json:"mean"`
	Rank       int     `json:"rank"`
	Popularity int     `json:"popularity"`
	NSFW       string  `json:"nsfw" jsonschema:"enum=white,enum=gray,enum=black"`
	GenreList  []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	MediaType         string `json:"media_type" jsonschema:"enum=unknown,enum=manga,enum=novel,enum=one_shot,enum=doujinshi,enum=manhwa,enum=manhua,enum=oel"`
	PublicationStatus string `json:"status" jsonschema:"enum=finished,enum=currently_publishing,enum=not_yet_published"`
	NumVolumes        int    `json:"num_volumes"`
	NumChapters       int    `json:"num_chapters"`
	AuthorList        []struct {
		Node struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"node"`
		Role string `json:"role"`
	} `json:"authors"`
}
