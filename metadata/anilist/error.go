package anilist

type Error string

func (e Error) Error() string {
	return "anilist error: " + e.Error()
}
