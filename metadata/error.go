package metadata

// Error is a general error for metadata operations.
type Error string

func (e Error) Error() string {
	return "metadata: " + string(e)
}
