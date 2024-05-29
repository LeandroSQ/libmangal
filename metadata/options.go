package metadata

// ComicInfoXMLOptions tweaks ComicInfoXML generation.
type ComicInfoXMLOptions struct {
	// AddDate whether to add series release date or not.
	AddDate bool

	// AlternativeDate use other date.
	AlternativeDate *Date
}

// DefaultComicInfoOptions constructs default ComicInfoXMLOptions.
func DefaultComicInfoOptions() ComicInfoXMLOptions {
	return ComicInfoXMLOptions{
		AddDate: true,
	}
}
