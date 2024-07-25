package libmangal

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

// Client is the wrapper around Provider with the extended functionality.
//
// It's the core of the libmangal.
type Client struct {
	provider Provider
	meta     map[metadata.IDCode]*metadata.ProviderWithCache
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
	provider.SetLogger(logger)

	return &Client{
		provider: provider,
		meta:     map[metadata.IDCode]*metadata.ProviderWithCache{},
		options:  options,
		logger:   logger,
	}, nil
}

func (c *Client) FS() afero.Fs {
	return c.options.FS
}

// AddMetadataProvider will add or update the metadata Provider.
func (c *Client) AddMetadataProvider(provider *metadata.ProviderWithCache) error {
	if provider == nil {
		return errors.New("Provider must be non-nil")
	}

	id := provider.Info().ID
	if id == "" {
		return errors.New("metadata Provider ID must be non-empty")
	}

	c.meta[id] = provider
	return nil
}

// GetMetadataProvider returns the requested metadata Provider for the given id.
func (c *Client) GetMetadataProvider(id metadata.IDCode) (*metadata.ProviderWithCache, error) {
	p, ok := c.meta[id]
	if !ok {
		return nil, errors.New("no metadata Provider found with ID " + string(id))
	}
	return p, nil
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
// 2. If the manga Title field is binded to an Anilist ID.
//
// 3. Otherwise find closest anilist manga (FindClosestManga) by using the manga Title field.
func (c *Client) SearchMetadata(
	ctx context.Context,
	manga mangadata.Manga,
) (metadata.Metadata, error) {
	c.logger.Log("searching metadata for manga %q on all available providers", manga)

	if len(c.meta) == 0 {
		return nil, errors.New("no metadata Providers available")
	}

	for id, p := range c.meta {
		meta, found, err := c.SearchByManga(ctx, p, manga)
		if err != nil {
			return nil, err
		}
		if !found {
			c.logger.Log("no metadata found for manga %q on metadata Provider %q", manga, string(id))
			continue
		}

		c.logger.Log("found metadata for manga %q on metadata Provider %q", manga, string(id))
		return meta, nil
	}

	return nil, nil
}

// SearchByManga is a convenience method to search given a Manga.
// It's meant to be used by the SearchMetadata method.
//
// Tries to search manga metadata in the following order:
//
// 1. If the manga contains non-nil metadata, by its metadata ID if available.
//
// 2 If the manga title is binded to a metadata ID.
//
// 3 Find closest manga metadata (FindClosest) by using the manga Title field.
func (c *Client) SearchByManga(ctx context.Context, provider *metadata.ProviderWithCache, manga mangadata.Manga) (metadata.Metadata, bool, error) {
	c.logger.Log("finding manga metadata by (libmangal) manga on %q", c.Info().Name)

	// Try to search by metadata ID if it is available
	meta := manga.Metadata()
	for _, id := range meta.ExtraIDs() {
		if id.Source == provider.Info().Source {
			anilistManga, found, err := provider.SearchByID(ctx, id.Value())
			if err == nil && found {
				return anilistManga, true, nil
			}
		}
	}

	// Else try to search by the title, this doesn't ensure
	// that the found manga metadata is 100% corresponding to
	// the manga requested, there are some instances in which
	// the result will be wrong
	title := manga.Info().Title
	return provider.FindClosest(ctx, title, 3, 3)
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
	// Found metadata will be replacing the incoming one,
	// even when no metadata is found (nil)
	if options.SearchMetadata {
		m, err := c.SearchMetadata(ctx, manga)
		if err != nil {
			return nil, err
		}
		manga.SetMetadata(m)
	}
	// Even after a metadata search, check if it is valid (nil for example)
	if err := metadata.Validate(manga.Metadata()); err != nil && options.Strict {
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

	if !options.SaveAnilist || !options.SaveHistory {
		return nil
	}

	if len(c.meta) == 0 {
		return errors.New("no metadata Providers available to mark chapter as read")
	}

	var setProgressErrors []error
	progress := int(math.Trunc(float64(chapter.Info().Number)))
	mangaTitle := chapter.Volume().Manga().Info().Title
	for id, p := range c.meta {
		metaID := 0
		// TODO: find a better way to get the metadata for the current provider
		meta, found, err := p.FindClosest(ctx, mangaTitle, 3, 3)
		if err != nil {
			goto addError
		}
		if !found {
			err = errors.New("no manga metadata found with Provider ID " + string(id))
			goto addError
		}
		metaID = meta.ID().Value()

		// TODO: add other providers
		switch {
		case id == metadata.IDCodeAnilist && options.SaveAnilist:
			err = p.SetMangaProgress(ctx, metaID, progress)
		case id == metadata.IDCodeMyAnimeList && options.SaveMyAnimeList:
		}
		if err != nil {
			goto addError
		}

		continue
	addError:
		c.logger.Log("error while setting manga progress for Provider ID %q: %s", string(id), err.Error())
		setProgressErrors = append(setProgressErrors, err)
	}

	// TODO: save to local history

	if len(setProgressErrors) != 0 {
		return errors.Join(setProgressErrors...)
	}
	return nil
}

// ProviderName determines the provider directory name.
func (c *Client) ProviderName(provider ProviderInfo) string {
	return c.options.ProviderName(provider)
}

// MangaName determines the manga directory name.
func (c *Client) MangaName(manga mangadata.Manga) string {
	return c.options.MangaName(c.Info(), manga)
}

// VolumeName determines the volume directory name.
// E.g. "Vol. 1" or "Volume 1"
func (c *Client) VolumeName(volume mangadata.Volume) string {
	return c.options.VolumeName(c.Info(), volume)
}

// ChapterName determines the chapter file name.
// E.g. "[001] chapter 1" or "Chainsaw Man - Ch. 1"
func (c *Client) ChapterName(chapter mangadata.Chapter, format Format) string {
	return c.options.ChapterName(c.Info(), chapter) + format.Extension()
}
