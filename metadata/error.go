package metadata

type Error string

func (e Error) Error() string {
	return "metadata error: " + string(e)
}
