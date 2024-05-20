package libmangal

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

// Client is the wrapper around Provider with the extended functionality.
//
// It's the core of the libmangal.
type Client struct {
	provider Provider
	options  ClientOptions
	logger   *Logger
}

// NewClient creates a new client from ProviderLoader.
//
// ClientOptions must be non-nil. Use DefaultClientOptions for defaults.
// It will validate ProviderLoader.Info and load the provider.
func NewClient(
	ctx context.Context,
	loader ProviderLoader,
	options ClientOptions,
) (*Client, error) {
	providerInfo := loader.Info()

	if err := providerInfo.Validate(); err != nil {
		return nil, err
	}

	provider, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	logger := NewLogger()
	logger.SetPrefix(providerInfo.ID)
	provider.SetLogger(logger)

	return &Client{
		provider: provider,
		options:  options,
		logger:   logger,
	}, nil
}

func (c *Client) FS() afero.Fs {
	return c.options.FS
}

func (c *Client) Anilist() *Anilist {
	return c.options.Anilist
}

func (c *Client) Logger() *Logger {
	return c.logger
}

func (c *Client) Close() error {
	return c.provider.Close()
}

// SearchMangas searches for mangas with the given query.
func (c *Client) SearchMangas(ctx context.Context, query string) ([]Manga, error) {
	return c.provider.SearchMangas(ctx, query)
}

// MangaVolumes gets chapters of the given manga.
func (c *Client) MangaVolumes(ctx context.Context, manga Manga) ([]Volume, error) {
	return c.provider.MangaVolumes(ctx, manga)
}

// VolumeChapters gets chapters of the given manga.
func (c *Client) VolumeChapters(ctx context.Context, volume Volume) ([]Chapter, error) {
	return c.provider.VolumeChapters(ctx, volume)
}

// ChapterPages gets pages of the given chapter.
func (c *Client) ChapterPages(ctx context.Context, chapter Chapter) ([]Page, error) {
	return c.provider.ChapterPages(ctx, chapter)
}

func (c *Client) String() string {
	return c.provider.Info().Name
}

// Info returns info about provider.
func (c *Client) Info() ProviderInfo {
	return c.provider.Info()
}

// TODO: add data about ComicInfoXML?
//
// DownloadedChapter provides a way to move downloaded chapter
// data around for easier handling.
type DownloadedChapter struct {
	// Name of the chapter, without directories.
	Name string `json:"name"`

	// Directory of the chapter (absolute).
	Directory string `json:"directory"`

	// ChapterStatus is the status of the downloaded chapter.
	ChapterStatus DownloadStatus `json:"chapter_status"`

	// SeriesJSONStatus is the status of the downloaded series.json.
	SeriesJSONStatus DownloadStatus `json:"series_json_status"`

	// ChapterStatus is the status of the downloaded chapter
	CoverStatus DownloadStatus `json:"cover_status"`

	// ChapterStatus is the status of the downloaded chapter.
	BannerStatus DownloadStatus `json:"banner_status"`
}

func (dc DownloadedChapter) Path() string {
	return filepath.Join(dc.Directory, dc.Name)
}

type DownloadStatus string

const (
	DownloadStatusNew         DownloadStatus = "new"
	DownloadStatusSkip        DownloadStatus = "skip"
	DownloadStatusExists      DownloadStatus = "exists"
	DownloadStatusOverwritten DownloadStatus = "overwritten"
)

// DownloadChapter downloads and saves chapter to the specified
// directory in the given format.
//
// It will return resulting chapter path joined with DownloadOptions.Directory
func (c *Client) DownloadChapter(
	ctx context.Context,
	chapter Chapter,
	options DownloadOptions,
) (DownloadedChapter, error) {
	c.logger.Log(fmt.Sprintf("Downloading chapter %q as %s", chapter, options.Format))

	// a temp client is used to download everything
	// into temp memory, then it is moved into the actual
	// location provided to the client
	tmpClient := Client{
		provider: c.provider,
		options:  c.options,
		logger:   c.logger,
	}

	tmpClient.options.FS = afero.NewMemMapFs()

	downChap, err := tmpClient.downloadChapterWithMetadata(ctx, chapter, options, func(path string) (bool, error) {
		return afero.Exists(c.options.FS, path)
	})
	if err != nil {
		return DownloadedChapter{}, err
	}

	if err := mergeDirectories(
		c.FS(), options.Directory,
		tmpClient.FS(), options.Directory,
	); err != nil {
		return DownloadedChapter{}, err
	}

	path := filepath.Join(downChap.Directory, downChap.Name)
	if options.ReadAfter {
		return downChap, c.ReadChapter(ctx, path, chapter, options.ReadOptions)
	}

	return downChap, nil
}

// DownloadPagesInBatch downloads multiple pages in batch
// by calling DownloadPage for each page in a separate goroutines.
//
// If any of the pages fails to download it will stop downloading other pages
// and return error immediately.
func (c *Client) DownloadPagesInBatch(
	ctx context.Context,
	pages []Page,
) ([]PageWithImage, error) {
	if len(pages) == 0 {
		return nil, fmt.Errorf("No pages provided for chapter")
	}
	c.logger.Log(fmt.Sprintf("Downloading %d pages", len(pages)))

	g, ctx := errgroup.WithContext(ctx)
	downloadedPages := make([]PageWithImage, len(pages))
	for i, page := range pages {
		g.Go(func() error {
			c.logger.Log(fmt.Sprintf("Page #%03d: downloading", i+1))

			downloaded, err := c.DownloadPage(ctx, page)
			if err != nil {
				return err
			}

			c.logger.Log(fmt.Sprintf("Page #%03d: done", i+1))

			downloadedPages[i] = downloaded
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return downloadedPages, nil
}

// DownloadPage downloads a page contents (image).
func (c *Client) DownloadPage(ctx context.Context, page Page) (PageWithImage, error) {
	if withImage, ok := page.(PageWithImage); ok {
		return withImage, nil
	}

	image, err := c.provider.GetPageImage(ctx, page)
	if err != nil {
		return nil, err
	}

	return &pageWithImage{
		Page:  page,
		image: image,
	}, nil
}

func (c *Client) ReadChapter(ctx context.Context, path string, chapter Chapter, options ReadOptions) error {
	c.logger.Log("Opening chapter with the default app")

	err := open.Run(path)
	if err != nil {
		return err
	}

	if options.SaveAnilist && c.Anilist().IsAuthorized() {
		return c.markChapterAsRead(ctx, chapter)
	}

	// TODO: save to local history

	return nil
}

func (c *Client) ComputeProviderFilename(provider ProviderInfo) string {
	return c.options.ProviderNameTemplate(provider)
}

func (c *Client) ComputeMangaFilename(manga Manga) string {
	return c.options.MangaNameTemplate(c.String(), manga)
}

func (c *Client) ComputeVolumeFilename(volume Volume) string {
	return c.options.VolumeNameTemplate(c.String(), volume)
}

func (c *Client) ComputeChapterFilename(chapter Chapter, format Format) string {
	return c.options.ChapterNameTemplate(c.String(), chapter) + format.Extension()
}
