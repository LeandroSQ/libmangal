package metadata

import "fmt"

type Error struct {
	E error
}

func (m Error) Error() string {
	return fmt.Sprintf("metadata error: %s", m.E.Error())
}
