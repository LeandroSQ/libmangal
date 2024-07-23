package myanimelist

type Error string

func (e Error) Error() string {
	return "mal: " + string(e)
}
