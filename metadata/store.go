package metadata

import (
	"strconv"

	"github.com/philippgille/gokv"
)

const (
	// QueryToIDs maps manga query to multiple metadata ids.
	//
	// ["berserk" => [7, 42, 69], "death note" => [887, 3, 134]]
	CacheBucketNameQueryToIDs = "query-to-ids"

	// TitleToID maps manga title to metadata id.
	//
	// ["berserk" => 7, "death note" => 3]
	CacheBucketNameTitleToID = "title-to-id"

	// IDToManga maps anilist id to metadata manga.
	//
	// [7 => "{title: ..., image: ..., ...}"]
	CacheBucketNameIDToManga = "id-to-manga"

	// AccessData maps username to access data (authentication).
	//
	// An User with the same name must be stored, too.
	CacheBucketNameNameToAccessData = "name-to-access-data"

	// NameToUser maps username to an user (authenticated user).
	//
	// An AccessData with the same name must be stored, too.
	CacheBucketNameNameToUser = "name-to-user"
)

type store struct {
	openStore func(bucketName string) (gokv.Store, error)
	store     gokv.Store
}

func (s *store) open(bucketName string) error {
	store, err := s.openStore(bucketName)
	s.store = store
	return err
}

func (s *store) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *store) getQueryIDs(query string) (ids []int, found bool, err error) {
	err = s.open(CacheBucketNameQueryToIDs)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(query, &ids)
	return
}

func (s *store) setQueryIDs(query string, ids []int) (err error) {
	err = s.open(CacheBucketNameQueryToIDs)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(query, ids)
}

func (s *store) getTitleID(title string) (id int, found bool, err error) {
	err = s.open(CacheBucketNameTitleToID)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(title, &id)
	return
}

func (s *store) setTitleID(title string, id int) (err error) {
	err = s.open(CacheBucketNameTitleToID)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(title, id)
}

func (s *store) getMeta(id int) (manga Metadata, found bool, err error) {
	err = s.open(CacheBucketNameIDToManga)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(strconv.Itoa(id), &manga)
	return
}

func (s *store) setMeta(id int, manga Metadata) (err error) {
	err = s.open(CacheBucketNameIDToManga)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(strconv.Itoa(id), &manga)
}

func (s *store) getAuthData(key string) (authData AuthData, found bool, err error) {
	err = s.open(CacheBucketNameNameToAccessData)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(key, &authData)
	return
}

func (s *store) setAuthData(key string, authData AuthData) (err error) {
	err = s.open(CacheBucketNameNameToAccessData)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(key, authData)
}

func (s *store) deleteAuthData(key string) (err error) {
	err = s.open(CacheBucketNameNameToAccessData)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Delete(key)
}

func (s *store) getUser(name string) (user User, found bool, err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	found, err = s.store.Get(name, &user)
	return
}

func (s *store) setUser(name string, user User) (err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Set(name, &user)
}

func (s *store) deleteUser(name string) (err error) {
	err = s.open(CacheBucketNameNameToUser)
	if err != nil {
		return
	}
	defer s.Close()

	return s.store.Delete(name)
}
