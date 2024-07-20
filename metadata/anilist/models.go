package anilist

type byIDData struct {
	Media *Manga `json:"media"`
}

type mangasData struct {
	Page struct {
		Media []Manga `json:"media"`
	} `json:"page"`
}

type setProgressData struct {
	SaveMediaListEntry struct {
		ID int `json:"id"`
	} `json:"SaveMediaListEntry"`
}

type userData struct {
	Viewer User `json:"viewer"`
}

// the token expires in a year, there shouldn't be need to handle refreshes...
type oAuthData struct {
	AccessToken string `json:"access_token"`
}
