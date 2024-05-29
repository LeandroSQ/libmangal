package mangadata

import "fmt"

// Page is what Chapter consists of.
type Page interface {
	fmt.Stringer

	// Extension gets the image extension of this page.
	// An extension must start with a dot.
	//
	// For example: .jpeg .png
	Extension() string

	// Chapter gets the Chapter that this Page is relevant to.
	//
	// Implementation should not make any external requests
	// nor be computationally heavy.
	Chapter() Chapter
}

// PageWithImage is a Page with already downloaded image.
//
// The associated image will be used instead of downloading one.
type PageWithImage interface {
	Page

	// Image gets the image contents.
	//
	// Implementation should not make any external requests.
	// Should only be exposed if the Page already contains image contents.
	Image() []byte

	// SetImage sets the image contents. This is used by DownloadOptions.ImageTransformer.
	SetImage(newImage []byte)
}
