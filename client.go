package libmangal

import (
	"context"

	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/spf13/afero"
)

// Client is a wrapper around Provider with extended functionality.
//
// It's the core of libmangal.
type Client struct {
	provider Provider
	meta     map[metadata.IDCode]*metadata.ProviderWithCache
	options  ClientOptions
	logger   *logger.Logger
}

// NewClient creates a new client from given ProviderLoader.
//
// ClientOptions must be non-zero. Use DefaultClientOptions for defaults.
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

func (c *Client) Close() error {
	return c.provider.Close()
}

func (c *Client) String() string {
	return c.provider.Info().Name
}

// Info returns info about provider.
func (c *Client) Info() ProviderInfo {
	return c.provider.Info()
}

// Logger returns the client's Logger.
func (c *Client) Logger() *logger.Logger {
	return c.logger
}

// FS returns the client's FileSystem.
func (c *Client) FS() afero.Fs {
	return c.options.FS
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
