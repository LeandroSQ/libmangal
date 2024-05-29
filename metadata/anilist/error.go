package anilist

import "fmt"

type Error struct {
	E error
}

func (a Error) Error() string {
	return fmt.Sprintf("anilist error: %s", a.E.Error())
}
