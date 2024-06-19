package metadata

import "fmt"

type Error struct {
	E error
}

func (e Error) Error() string {
	return fmt.Sprintf("metadata error: %s", e.E.Error())
}
