package anilist

type Error string

func (e Error) Error() string {
	return "anilist error: " + string(e)
}

type CodeGrantError string

func (e CodeGrantError) Error() string {
	return "code grant error: " + string(e)
}

type AuthError string

func (e AuthError) Error() string {
	return "anilist auth error: " + string(e)
}
