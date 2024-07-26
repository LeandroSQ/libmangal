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
