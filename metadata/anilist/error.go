package anilist

type Error string

func (e Error) Error() string {
	return "anilist error: " + string(e)
}

type LoginCredentialsError string

func (e LoginCredentialsError) Error() string {
	return "anilist login credentials error: " + string(e)
}
