package metadata

// Error is a general error for metadata operations.
type Error string

func (e Error) Error() string {
	return "metadata: " + string(e)
}

type AuthError string

func (e AuthError) Error() string {
	return "auth: " + string(e)
}

type CodeGrantError string

func (e CodeGrantError) Error() string {
	return "code grant: " + string(e)
}
