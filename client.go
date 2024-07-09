package libmangal

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/libmangal/metadata/anilist"
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
	logger   *logger.Logger
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

	logger := logger.NewLogger()
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

func (c *Client) Anilist() *anilist.Anilist {
	return c.options.Anilist
}

func (c *Client) Logger() *logger.Logger {
	return c.logger
}

func (c *Client) Close() error {
	return c.provider.Close()
}

// SearchMangas searches for mangas with the given query.
func (c *Client) SearchMangas(ctx context.Context, query string) ([]mangadata.Manga, error) {
	return c.provider.SearchMangas(ctx, query)
}

// MangaVolumes gets chapters of the given manga.
func (c *Client) MangaVolumes(ctx context.Context, manga mangadata.Manga) ([]mangadata.Volume, error) {
	return c.provider.MangaVolumes(ctx, manga)
}

// VolumeChapters gets chapters of the given manga.
func (c *Client) VolumeChapters(ctx context.Context, volume mangadata.Volume) ([]mangadata.Chapter, error) {
	return c.provider.VolumeChapters(ctx, volume)
}

// ChapterPages gets pages of the given chapter.
func (c *Client) ChapterPages(ctx context.Context, chapter mangadata.Chapter) ([]mangadata.Page, error) {
	return c.provider.ChapterPages(ctx, chapter)
}

func (c *Client) String() string {
	return c.provider.Info().Name
}

// Info returns info about provider.
func (c *Client) Info() ProviderInfo {
	return c.provider.Info()
}

// SearchMetadata will search for metadata on the available metadata providers.
//
// Tries to search anilist manga in the following order:
//
// 1. If the manga contains non-nil metadata, by its Anilist ID if available.
//
// 2. If the manga title (priority on AnilistSearch field, then Title field) is binded to an Anilist ID.
//
// 3. Otherwise find closest anilist manga (FindClosestManga) by using the manga Title (priority on AnilistSearch field) field.
func (c *Client) SearchMetadata(
	ctx context.Context,
	manga mangadata.Manga,
) (*metadata.Metadata, error) {
	c.logger.Log("searching metadata for manga %q", manga)
	anilistManga, found, err := c.Anilist().SearchByManga(ctx, manga)
	if err != nil {
		return nil, err
	}
	if !found {
		c.logger.Log("couldn't find associated anilist manga for %q", manga)
		return nil, nil
	}

	return anilistManga.Metadata(), nil
}

// DownloadChapter downloads and saves chapter to the specified
// directory in the given format.
//
// It will return resulting chapter downloaded information via metadata.DownloadedChapter.
func (c *Client) DownloadChapter(
	ctx context.Context,
	chapter mangadata.Chapter,
	options DownloadOptions,
) (*metadata.DownloadedChapter, error) {
	c.logger.Log("downloading chapter %q as %s", chapter, options.Format)

	manga := chapter.Volume().Manga()
	// Found metadata will be replacing the incoming one, even when no metadata is found (nil)
	if options.SearchMetadata {
		m, err := c.SearchMetadata(ctx, manga)
		if err != nil {
			return nil, err
		}
		manga.SetMetadata(m)
	}
	// Even after a metadata search, check if it is valid (nil for example)
	if err := manga.Metadata().Validate(); err != nil && options.Strict {
		return nil, fmt.Errorf("no valid metadata for manga %q: %s", manga, err.Error())
	}

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
		return nil, err
	}

	if err := mergeDirectories(
		c.options.ModeDir,
		c.FS(), options.Directory,
		tmpClient.FS(), options.Directory,
	); err != nil {
		return nil, err
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
	pages []mangadata.Page,
) ([]mangadata.PageWithImage, error) {
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages provided for chapter")
	}
	c.logger.Log("downloading %d pages", len(pages))

	g, ctx := errgroup.WithContext(ctx)
	downloadedPages := make([]mangadata.PageWithImage, len(pages))
	for i, page := range pages {
		g.Go(func() error {
			c.logger.Log("page #%03d: downloading", i+1)

			downloaded, err := c.DownloadPage(ctx, page)
			if err != nil {
				return err
			}

			c.logger.Log("page #%03d: done", i+1)

			downloadedPages[i] = downloaded
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return downloadedPages, nil
}

type pageWithImage struct {
	mangadata.Page
	image []byte
}

func (p *pageWithImage) Image() []byte {
	return p.image
}

func (p *pageWithImage) SetImage(newImage []byte) {
	p.image = newImage
}

// DownloadPage downloads a page contents (image).
func (c *Client) DownloadPage(
	ctx context.Context,
	page mangadata.Page,
) (mangadata.PageWithImage, error) {
	if withImage, ok := page.(mangadata.PageWithImage); ok {
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

// ReadChapter opens the chapter for reading and marks it as read if authorized.
// It will use os default app for resulting mimetype.
//
// E.g. `xdg-open` for Linux.
//
// It will also sync read chapter with your Anilist profile
// if it's configured.
//
// Note, that underlying filesystem must be mapped with OsFs
// in order for os to open it.
func (c *Client) ReadChapter(
	ctx context.Context,
	path string,
	chapter mangadata.Chapter,
	options ReadOptions,
) error {
	c.logger.Log("opening chapter %q from %s with the default app", chapter, path)

	err := open.Run(path)
	if err != nil {
		return err
	}

	// TODO: generalize this to mark as read on multiple metadata providers
	if options.SaveAnilist && c.Anilist().IsAuthorized() {
		return c.markChapterAsRead(ctx, chapter)
	}

	// TODO: save to local history

	return nil
}

func (c *Client) ComputeProviderFilename(provider ProviderInfo) string {
	return c.options.ProviderNameTemplate(provider)
}

func (c *Client) ComputeMangaFilename(manga mangadata.Manga) string {
	return c.options.MangaNameTemplate(c.String(), manga)
}

func (c *Client) ComputeVolumeFilename(volume mangadata.Volume) string {
	return c.options.VolumeNameTemplate(c.String(), volume)
}

func (c *Client) ComputeChapterFilename(chapter mangadata.Chapter, format Format) string {
	return c.options.ChapterNameTemplate(c.String(), chapter) + format.Extension()
}
