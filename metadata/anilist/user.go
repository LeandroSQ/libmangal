package anilist

// User is the user model for Anilist.
type User struct {
	ID     int    `json:"id" jsonschema:"description=The ID of the user."`
	Name   string `json:"name" jsonschema:"description=The name of the user."`
	About  string `json:"about" jsonschema:"description=The bio written by user (Markdown)."`
	Avatar struct {
		Large  string `json:"large" jsonschema:"description=The avatar of user at its largest size."`
		Medium string `json:"medium" jsonschema:"description=The avatar of user at medium size."`
	} `json:"avatar" jsonschema:"description=The user's avatar images."`
	BannerImage string `json:"bannerImage" jsonschema:"description=The user's banner images."`
	Options     struct {
		TitleLanguage       string `json:"titleLanguage" jsonschema:"enum=ROMAJI,enum=ENGLISH,enum=NATIVE,enum=ROMAJI_STYLISED,enum=ENGLISH_STYLISED,enum=NATIVE_STYLISED"`
		DisplayAdultContent bool   `json:"displayAdultContent" jsonschema:"description=Whether the user has enabled viewing of 18+ content."`
		ProfileColor        string `json:"profileColor" jsonschema:"description=Profile highlight color (blue, purple, pink, orange, red, green, gray)."`
		Timezone            string `json:"timezone" jsonschema:"description=The user's timezone offset (Auth user only)."`
	} `json:"options" jsonschema:"The user's general options."`
	SiteURL       string `json:"siteUrl" jsonschema:"description=The URL for the user page on the AniList website."`
	CreatedAt     int    `json:"createdAt" jsonschema:"description=When the user's account was created. (Does not exist for accounts created before 2020)."`
	UpdatedAt     int    `json:"updatedAt" jsonschema:"description=When the user's data was last updated."`
	PreviousNames []struct {
		Name      string `json:"name" jsonschema:"description=The name of the user."`
		CreatedAt int    `json:"createdAt" jsonschema:"description=When the user first changed from this name."`
		UpdatedAt int    `json:"updatedAt" jsonschema:"description=When the user most recently changed from this name."`
	} `json:"previous_names" jsonschema:"description=The user's previously used names."`
}
