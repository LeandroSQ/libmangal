package anilist

import "fmt"

type Error struct {
	E error
}

func (e Error) Error() string {
	return fmt.Sprintf("anilist error: %s", e.E.Error())
}
