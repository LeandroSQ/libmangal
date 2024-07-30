package anilist

// Error is a general error for Anilist operations.
type Error string

func (e Error) Error() string {
	return "anilist: " + string(e)
}
