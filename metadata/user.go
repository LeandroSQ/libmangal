package metadata

// User is the general authenticated user information.
type User interface {
	// String is the short representation of the user.
	// Must be non-empty.
	//
	// For example "`Name` (`ID`)".
	String() string

	// ID is the id of the user.
	ID() int

	// Name of the user.
	Name() string

	// About is the about section of the user.
	About() string

	// Avatar is the URL of the avatar image.
	Avatar() string

	// URL is the user's URL on the metadata provider website.
	URL() string

	// Source provider of the user.
	//
	// For example if coming from Anilist: IDSourceAnilist.
	Source() IDSource
}
