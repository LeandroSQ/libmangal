package anilist

import "strconv"

func (a *Anilist) cacheStatusQuery(
	query string,
) (ids []int, found bool, err error) {
	found, err = a.options.QueryToIDsStore.Get(query, &ids)
	return
}

func (a *Anilist) cacheSetQuery(
	query string,
	ids []int,
) error {
	return a.options.QueryToIDsStore.Set(query, ids)
}

func (a *Anilist) cacheStatusTitle(
	title string,
) (id int, found bool, err error) {
	found, err = a.options.TitleToIDStore.Get(title, &id)
	return
}

func (a *Anilist) cacheSetTitle(
	title string,
	id int,
) error {
	return a.options.TitleToIDStore.Set(title, id)
}

func (a *Anilist) cacheStatusID(
	id int,
) (manga Manga, found bool, err error) {
	found, err = a.options.IDToMangaStore.Get(strconv.Itoa(id), &manga)
	return
}

func (a *Anilist) cacheSetID(
	id int,
	manga Manga,
) error {
	return a.options.IDToMangaStore.Set(strconv.Itoa(id), manga)
}
