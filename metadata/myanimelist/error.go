package myanimelist

// Error is a general error for MyAnimeList operations.
type Error string

func (e Error) Error() string {
	return "mal: " + string(e)
}
