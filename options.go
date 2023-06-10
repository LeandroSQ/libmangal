package libmangal

import (
	"fmt"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
	"github.com/spf13/afero"
	"net/http"
)

// DownloadOptions configures Chapter downloading
type DownloadOptions struct {
	// Format in which a chapter must be downloaded
	Format Format

	// CreateMangaDir will create manga directory
	CreateMangaDir bool

	// CreateVolumeDir will create volume directory.
	//
	// If CreateMangaDir is also true, volume directory
	// will be created under it.
	CreateVolumeDir bool

	// Strict means that that if during metadata creation
	// error occurs downloader will return it immediately and chapter
	// won't be downloaded
	Strict bool

	// SkipIfExists will skip downloading chapter if its already downloaded (exists)
	SkipIfExists bool

	// DownloadMangaCover or not. Will not download cover again if its already downloaded.
	DownloadMangaCover bool

	// WriteSeriesJson write metadata series.json file in the manga directory
	WriteSeriesJson bool

	// WriteComicInfoXml write metadata ComicInfo.xml file to the .cbz archive when
	// downloading with FormatCBZ
	WriteComicInfoXml bool

	// ComicInfoOptions options to use for ComicInfo.xml when WriteComicInfoXml is true
	ComicInfoOptions ComicInfoXmlOptions
}

// DefaultDownloadOptions constructs default DownloadOptions
func DefaultDownloadOptions() DownloadOptions {
	return DownloadOptions{
		Format:             FormatPDF,
		CreateMangaDir:     true,
		CreateVolumeDir:    false,
		Strict:             true,
		SkipIfExists:       true,
		DownloadMangaCover: false,
		WriteSeriesJson:    false,
		WriteComicInfoXml:  false,
		ComicInfoOptions:   DefaultComicInfoOptions(),
	}
}

// ReadOptions specifies reading options passed to the Client.ReadChapter
type ReadOptions struct {
	// Format used for reading
	Format Format

	// MangasLibraryPath is the path to the directory where mangas are stored.
	// Will be used to see if the given chapter is already downloaded,
	// so it will be opened instead
	MangasLibraryPath string
}

// DefaultReadOptions constructs default ReadOptions
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		Format:            FormatPDF,
		MangasLibraryPath: "",
	}
}

// AnilistOptions is options for Anilist client
type AnilistOptions struct {
	// HTTPClient is a http client used for Anilist API
	HTTPClient *http.Client

	// QueryToIDsStore maps query to ids.
	// single query to multiple ids.
	//
	// ["berserk" => [7, 42, 69], "death note" => [887, 3, 134]]
	QueryToIDsStore gokv.Store

	// TitleToIDStore maps title to id.
	// single title to single id.
	//
	// ["berserk" => 7, "death note" => 3]
	TitleToIDStore gokv.Store

	// IDToMangaStore maps id to manga.
	// single id to single manga.
	//
	// [7 => "{title: ..., image: ..., ...}"]
	IDToMangaStore gokv.Store

	// Log logs progress
	Log LogFunc
}

// DefaultAnilistOptions constructs default AnilistOptions
func DefaultAnilistOptions() AnilistOptions {
	return AnilistOptions{
		Log: func(string) {},

		HTTPClient: &http.Client{},

		QueryToIDsStore: syncmap.NewStore(syncmap.DefaultOptions),
		TitleToIDStore:  syncmap.NewStore(syncmap.DefaultOptions),
		IDToMangaStore:  syncmap.NewStore(syncmap.DefaultOptions),
	}
}

// ClientOptions is options that client would use during its runtime.
// See DefaultClientOptions
type ClientOptions struct {
	// HTTPClient is http client that client would use for requests
	HTTPClient *http.Client

	// FS is a file system abstraction
	// that the client will use.
	FS afero.Fs

	// ChapterNameTemplate defines how mangas filenames will look when downloaded.
	MangaNameTemplate func(
		provider string,
		manga Manga,
	) string

	// ChapterNameTemplate defines how volumes filenames will look when downloaded.
	// E.g. Vol. 1
	VolumeNameTemplate func(
		provider string,
		volume Volume,
	) string

	// ChapterNameTemplate defines how chapters filenames will look when downloaded.
	// E.g. "[001] chapter 1" or "Chainsaw Man - Ch. 1"
	ChapterNameTemplate func(
		provider string,
		chapter Chapter,
	) string

	// Log is a function that will be passed to the provider
	// to serve as a progress writer
	Log LogFunc

	// Anilist is the Anilist client to use
	Anilist Anilist
}

// DefaultClientOptions constructs default ClientOptions
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		HTTPClient: &http.Client{},
		FS:         afero.NewOsFs(),
		ChapterNameTemplate: func(_ string, chapter Chapter) string {
			info := chapter.Info()
			number := fmt.Sprintf("%06.1f", info.Number)
			return sanitizePath(fmt.Sprintf("[%s] %s", number, info.Title))
		},
		MangaNameTemplate: func(_ string, manga Manga) string {
			return sanitizePath(manga.Info().Title)
		},
		VolumeNameTemplate: func(_ string, volume Volume) string {
			return sanitizePath(fmt.Sprintf("Vol. %d", volume.Info().Number))
		},
		Log:     func(string) {},
		Anilist: NewAnilist(DefaultAnilistOptions()),
	}
}

// ComicInfoXmlOptions tweaks ComicInfoXml generation
type ComicInfoXmlOptions struct {
	// AddDate whether to add series release date or not
	AddDate bool

	// AlternativeDate use other date
	AlternativeDate *Date
}

// DefaultComicInfoOptions constructs default ComicInfoXmlOptions
func DefaultComicInfoOptions() ComicInfoXmlOptions {
	return ComicInfoXmlOptions{
		AddDate: true,
	}
}
