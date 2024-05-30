package libmangal

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/libmangal/metadata/anilist"
	"github.com/spf13/afero"
)

const (
	defaultUserAgent string      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0"
	defaultModeDir   fs.FileMode = 0o755
	defaultModeFile  fs.FileMode = 0o644
)

// ReadOptions configures the reader options.
type ReadOptions struct {
	// SaveHistory will save chapter to local history if ReadAfter is enabled.
	SaveHistory bool

	// SaveAnilist will save Anilist reading history if ReadAfter is enabled and logged in to the Anilist.
	SaveAnilist bool
}

// DefaultReadOptions constructs default ReadOptions.
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		SaveHistory: false,
		SaveAnilist: false,
	}
}

// DownloadOptions configures Chapter downloading.
type DownloadOptions struct {
	// Format in which a chapter must be downloaded.
	Format Format

	// Directory is the directory where manga will be downloaded to.
	Directory string

	// CreateProviderDir will create provider directory.
	CreateProviderDir bool

	// CreateMangaDir will create manga directory.
	CreateMangaDir bool

	// CreateVolumeDir will create volume directory.
	//
	// If CreateMangaDir is also true, volume directory
	// will be created under it.
	CreateVolumeDir bool

	// Strict means that that if there is missing metadata or if an error occurs during
	// metadata files creation, the chapter will not be written to disk.
	//
	// Some metadata is potentially written to disk.
	//
	// Checks if the metadata is non-nil as well as if it contains the minimum metadata fields.
	// This check comes after SearchMissingMetadata.
	Strict bool

	// SkipIfExists will skip downloading chapter if its already downloaded (exists at path).
	//
	// However, metadata will still be created if needed.
	SkipIfExists bool

	// SearchMissingMetadata will search for missing metadata on the available metadata providers.
	// Checks if the metadata is non-nil as well as if it contains the minimum metadata fields.
	//
	// Search priority is always by ID (if provided as part of one of the metadata fields), then by the title.
	SearchMissingMetadata bool

	// DownloadMangaCover or not. Will not download cover again if its already downloaded.
	DownloadMangaCover bool

	// DownloadMangaBanner or not. Will not download banner again if its already downloaded.
	DownloadMangaBanner bool

	// WriteSeriesJSON write metadata series.json file in the manga directory.
	WriteSeriesJSON bool

	// SkipSeriesJSONIfOngoing will avoid writing series.json file for ongoing series,
	// due to lack of TotalIssues metadata provided by Anilist for example.
	//
	// There are issues with _some_ parsers when the TotalIssues is zero (I see you, Komga),
	// so this is a workaround. Also this avoids checks to overwrite series.json on each chapter,
	// silly, no (if only Komga would fix that)?
	SkipSeriesJSONIfOngoing bool

	// WriteComicInfoXML write metadata ComicInfo.xml file to the .cbz archive when
	// downloading with FormatCBZ.
	WriteComicInfoXML bool

	// ReadAfter will open the chapter for reading after it was downloaded.
	// It will use os default app for resulting mimetype.
	//
	// E.g. `xdg-open` for Linux.
	//
	// It will also sync read chapter with your Anilist profile
	// if it's configured. See also ReadIncognito.
	//
	// Note, that underlying filesystem must be mapped with OsFs
	// in order for os to open it.
	ReadAfter bool

	ReadOptions ReadOptions

	// ComicInfoXMLOptions options to use for ComicInfo.xml when WriteComicInfoXml is true.
	ComicInfoXMLOptions metadata.ComicInfoXMLOptions

	// ImageTransformer is applied for each image for the chapter.
	//
	// E.g. grayscale effect.
	ImageTransformer func([]byte) ([]byte, error)
}

// DefaultDownloadOptions constructs default DownloadOptions.
func DefaultDownloadOptions() DownloadOptions {
	return DownloadOptions{
		Format:                  FormatCBZ,
		Directory:               ".",
		CreateProviderDir:       false,
		CreateMangaDir:          true,
		CreateVolumeDir:         false,
		Strict:                  true,
		SkipIfExists:            true,
		SearchMissingMetadata:   true,
		DownloadMangaCover:      false,
		DownloadMangaBanner:     false,
		WriteSeriesJSON:         false,
		SkipSeriesJSONIfOngoing: true, // Sensible default to avoid external parser issues.
		WriteComicInfoXML:       false,
		ReadAfter:               false,
		ImageTransformer: func(img []byte) ([]byte, error) {
			return img, nil
		},
		ReadOptions:         DefaultReadOptions(),
		ComicInfoXMLOptions: metadata.DefaultComicInfoOptions(),
	}
}

// ClientOptions is options that client would use during its runtime.
type ClientOptions struct {
	// HTTPClient is http client that client would use for requests.
	HTTPClient *http.Client

	// UserAgent to use when making HTTP requests.
	UserAgent string

	// FS is a file system abstraction that the client will use.
	FS afero.Fs

	// ModeDir is the permission bits used for all dirs created.
	ModeDir fs.FileMode

	// ModeFile is the permission bits used for all files created.
	ModeFile fs.FileMode

	// ProviderNameTemplate defines how provider filenames will look when downloaded.
	ProviderNameTemplate func(
		provider ProviderInfo,
	) string

	// ChapterNameTemplate defines how mangas filenames will look when downloaded.
	MangaNameTemplate func(
		provider string,
		manga mangadata.Manga,
	) string

	// ChapterNameTemplate defines how volumes filenames will look when downloaded.
	// E.g. Vol. 1
	VolumeNameTemplate func(
		provider string,
		volume mangadata.Volume,
	) string

	// ChapterNameTemplate defines how chapters filenames will look when downloaded.
	// E.g. "[001] chapter 1" or "Chainsaw Man - Ch. 1"
	ChapterNameTemplate func(
		provider string,
		chapter mangadata.Chapter,
	) string

	// Anilist is the Anilist client to use.
	Anilist *anilist.Anilist
}

// DefaultClientOptions constructs default ClientOptions, with default Anilist options as well.
func DefaultClientOptions() ClientOptions {
	anilist := anilist.NewAnilist(anilist.DefaultOptions())
	return ClientOptions{
		HTTPClient: &http.Client{},
		UserAgent:  defaultUserAgent,
		ModeDir:    defaultModeDir,
		ModeFile:   defaultModeFile,
		FS:         afero.NewOsFs(),
		ProviderNameTemplate: func(provider ProviderInfo) string {
			return sanitizePath(provider.Name)
		},
		MangaNameTemplate: func(_ string, manga mangadata.Manga) string {
			return sanitizePath(manga.Info().Title)
		},
		VolumeNameTemplate: func(_ string, volume mangadata.Volume) string {
			return sanitizePath(fmt.Sprintf("Vol. %.1f", volume.Info().Number))
		},
		ChapterNameTemplate: func(_ string, chapter mangadata.Chapter) string {
			info := chapter.Info()
			number := fmt.Sprintf("%06.1f", info.Number)
			return sanitizePath(fmt.Sprintf("[%s] %s", number, info.Title))
		},
		Anilist: &anilist,
	}
}
