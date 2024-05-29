package mangadata

import "fmt"

// VolumeInfo is the general information for the volume.
type VolumeInfo struct {
	// Number of the volume.
	Number float32 `json:"number"`
}

// Volume of a manga. If a series is popular enough, its chapters
// are then collected and published into volumes,
// which usually feature a few chapters of the overall story.
// Most Manga series are long-running and can span multiple volumes.
//
// At least one volume is expected.
type Volume interface {
	fmt.Stringer

	Info() VolumeInfo

	// Manga gets the Manga that this Volume is relevant to.
	//
	// Implementation should not make any external requests
	// nor be computationally heavy.
	Manga() Manga
}
