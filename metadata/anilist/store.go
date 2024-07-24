package anilist

import (
	"fmt"

	"github.com/luevano/libmangal/metadata"
	"github.com/philippgille/gokv"
)

const (
	// AccessToken maps username to access tokens (authentication).
	//
	// An User with the same name must be stored, too.
	CacheBucketNameNameToAccessToken = "name-to-access-token"

	// NameToUser maps username to anilist user (authenticated user).
	//
	// An Accesstoken with the same name must be stored, too.
	CacheBucketNameNameToUser = "name-to-user"
)

type store struct {
	openStore func(bucketName string) (gokv.Store, error)
	store     gokv.Store
}

func (s *store) open(bucketName string) error {
	store, err := s.openStore(bucketName)
	if store == nil {
		fmt.Println("store is nil when opening bucket " + bucketName)
	}
	s.store = store
	if s.store == nil {
		fmt.Println("assigned store is nil when opening bucket " + bucketName)
	}
	return err
}

func (s *store) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *store) getAuthToken(key string) (token string, found bool, err error) {
	err = s.open(CacheBucketNameNameToAccessToken)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(key, &token)
	return
}

func (s *store) setAuthToken(key, authToken string) (err error) {
	err = s.open(CacheBucketNameNameToAccessToken)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(key, authToken)
}

func (s *store) deleteAuthToken(key string) (err error) {
	err = s.open(CacheBucketNameNameToAccessToken)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Delete(key)
}

func (s *store) getUser(name string) (user metadata.User, found bool, err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(name, &user)
	return
}

func (s *store) setUser(name string, user metadata.User) (err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(name, user)
}

func (s *store) deleteUser(name string) (err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Delete(name)
}
