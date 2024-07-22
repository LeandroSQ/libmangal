package anilist

type Error string

func (e Error) Error() string {
	return "anilist: " + string(e)
}

type AuthError string

func (e AuthError) Error() string {
	return "anilist auth: " + string(e)
}

type LogoutError string

func (e LogoutError) Error() string {
	return "anilist logout: " + string(e)
}

type CodeGrantError string

func (e CodeGrantError) Error() string {
	return "code grant: " + string(e)
}
