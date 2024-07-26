package myanimelist

import (
	"context"
	"errors"

	"github.com/luevano/libmangal/metadata"
)

// FIX: implement
//
// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *MyAnimeList) Authenticated() bool {
	return false
}

// FIX: implement
//
// Login authorizes an user with the given LoginOption.
func (p *MyAnimeList) Login(ctx context.Context, loginOption metadata.LoginOption) error {
	return errors.ErrUnsupported
}

// FIX: implement
//
// Logout de-authorizes the currently authorized user.
func (p *MyAnimeList) Logout() error {
	return errors.ErrUnsupported
}
