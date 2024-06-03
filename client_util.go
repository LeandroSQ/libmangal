package libmangal

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/spf13/afero"
)

type mangaImage string

const (
	mangaImageCover  mangaImage = "cover"
	mangaImageBanner mangaImage = "banner"
)

type pathExistsFunc func(string) (bool, error)

// removeChapter will remove chapter at given path.
//
// Doesn't matter if it's a directory or a file.
//
// Currently unused.
func (c *Client) removeChapter(chapterPath string) error {
	c.logger.Log("removing %s", chapterPath)

	isDir, err := afero.IsDir(c.options.FS, chapterPath)
	if err != nil {
		return err
	}

	if isDir {
		return c.options.FS.RemoveAll(chapterPath)
	}

	return c.options.FS.Remove(chapterPath)
}

func (c *Client) downloadChapterWithMetadata(
	ctx context.Context,
	chapter mangadata.Chapter,
	options DownloadOptions,
	existsFunc pathExistsFunc,
) (*metadata.DownloadedChapter, error) {
	directory := options.Directory

	var (
		seriesJSONDir = directory
		coverDir      = directory
		bannerDir     = directory
	)

	if options.CreateProviderDir {
		directory = filepath.Join(directory, c.ComputeProviderFilename(c.provider.Info()))
	}

	if options.CreateMangaDir {
		directory = filepath.Join(directory, c.ComputeMangaFilename(chapter.Volume().Manga()))
		seriesJSONDir = directory
		coverDir = directory
		bannerDir = directory
	}

	if options.CreateVolumeDir {
		directory = filepath.Join(directory, c.ComputeVolumeFilename(chapter.Volume()))
	}

	err := c.options.FS.MkdirAll(directory, c.options.ModeDir)
	if err != nil {
		return nil, err
	}

	chapterFilename := c.ComputeChapterFilename(chapter, options.Format)
	chapterPath := filepath.Join(directory, chapterFilename)

	chapterExists, err := existsFunc(chapterPath)
	if err != nil {
		return nil, err
	}

	manga := chapter.Volume().Manga()
	// Data about downloaded chapter
	downChap := &metadata.DownloadedChapter{
		Number:             chapter.Info().Number,
		Title:              chapter.Info().Title,
		Filename:           chapterFilename,
		Directory:          directory,
		ChapterStatus:      metadata.DownloadStatusExists,
		SeriesJSONStatus:   metadata.DownloadStatusSkip,
		ComicInfoXMLStatus: metadata.DownloadStatusSkip, // only CBZ writes it
		CoverStatus:        metadata.DownloadStatusSkip,
		BannerStatus:       metadata.DownloadStatusSkip,
	}

	if !chapterExists || !options.SkipIfExists {
		ciXmlStatus, err := c.downloadChapter(ctx, chapter, chapterPath, options)
		if err != nil {
			return nil, err
		}
		downChap.ComicInfoXMLStatus = ciXmlStatus

		downChap.ChapterStatus = metadata.DownloadStatusNew
		if !options.SkipIfExists {
			downChap.ChapterStatus = metadata.DownloadStatusOverwritten
		}
	}

	if manga.Metadata().Validate() != nil {
		downChap.SeriesJSONStatus = metadata.DownloadStatusMissingMetadata
		downChap.CoverStatus = metadata.DownloadStatusMissingMetadata
		downChap.BannerStatus = metadata.DownloadStatusMissingMetadata
		return downChap, nil
	}

	skip := options.SkipSeriesJSONIfOngoing && manga.Metadata().Status == metadata.StatusReleasing
	if options.WriteSeriesJSON && !skip {
		path := filepath.Join(seriesJSONDir, metadata.FilenameSeriesJSON)
		exists, err := existsFunc(path)
		if err != nil {
			return nil, err
		}

		downChap.SeriesJSONStatus = metadata.DownloadStatusExists
		if !exists {
			file, err := c.options.FS.Create(path)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			err = c.writeSeriesJSON(manga, file)
			downChap.SeriesJSONStatus = metadata.DownloadStatusNew
			if err != nil {
				downChap.SeriesJSONStatus = metadata.DownloadStatusFailed
				if options.Strict {
					return nil, metadata.Error{err}
				}
			}
		}
	}

	if options.DownloadMangaCover {
		path := filepath.Join(coverDir, metadata.FilenameCoverJPG)
		exists, err := existsFunc(path)
		if err != nil {
			return nil, err
		}

		downChap.CoverStatus = metadata.DownloadStatusExists
		if !exists {
			file, err := c.options.FS.Create(path)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			err = c.downloadMangaImage(ctx, manga, mangaImageCover, file)
			downChap.CoverStatus = metadata.DownloadStatusNew
			if err != nil {
				downChap.CoverStatus = metadata.DownloadStatusFailed
				if options.Strict {
					return nil, metadata.Error{err}
				}
			}
		}
	}

	if options.DownloadMangaBanner {
		path := filepath.Join(bannerDir, metadata.FilenameBannerJPG)
		exists, err := existsFunc(path)
		if err != nil {
			return nil, err
		}

		downChap.BannerStatus = metadata.DownloadStatusExists
		if !exists {
			file, err := c.options.FS.Create(path)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			err = c.downloadMangaImage(ctx, manga, mangaImageBanner, file)
			downChap.BannerStatus = metadata.DownloadStatusNew
			if err != nil {
				downChap.BannerStatus = metadata.DownloadStatusFailed
				if options.Strict {
					return nil, metadata.Error{err}
				}
			}
		}
	}

	return downChap, nil
}

// downloadChapter is a helper function for DownloadChapter
func (c *Client) downloadChapter(
	ctx context.Context,
	chapter mangadata.Chapter,
	path string,
	options DownloadOptions,
) (metadata.DownloadStatus, error) {
	pages, err := c.ChapterPages(ctx, chapter)
	if err != nil {
		return "", err
	}

	downloadedPages, err := c.DownloadPagesInBatch(ctx, pages)
	if err != nil {
		return "", err
	}

	for _, page := range downloadedPages {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		image, err := options.ImageTransformer(page.Image())
		if err != nil {
			return "", err
		}

		page.SetImage(image)
	}

	// Only CBZ writes the ComicInfo.xml, so by default it's skipped
	ciXmlStatusSkip := metadata.DownloadStatusSkip
	switch options.Format {
	case FormatPDF:
		file, err := c.options.FS.Create(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		return ciXmlStatusSkip, c.savePDF(downloadedPages, file)
	case FormatTAR:
		file, err := c.options.FS.Create(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		return ciXmlStatusSkip, c.saveTAR(downloadedPages, file)
	case FormatTARGZ:
		file, err := c.options.FS.Create(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		return ciXmlStatusSkip, c.saveTARGZ(downloadedPages, file)
	case FormatZIP:
		file, err := c.options.FS.Create(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		return ciXmlStatusSkip, c.saveZIP(downloadedPages, file)
	case FormatCBZ:
		var comicInfoXML *metadata.ComicInfoXML
		if options.WriteComicInfoXML && chapter.Volume().Manga().Metadata().Validate() == nil {
			mangaChapter := chapter.Info()
			metaChapter := metadata.Chapter{
				Title:           mangaChapter.Title,
				URL:             mangaChapter.URL,
				Number:          mangaChapter.Number,
				Date:            mangaChapter.Date,
				ScanlationGroup: mangaChapter.ScanlationGroup,
				Pages:           len(downloadedPages),
			}
			ciXML, err := c.getComicInfoXML(chapter, metaChapter)
			if err != nil && options.Strict {
				return "", err
			}
			comicInfoXML = &ciXML
		}

		file, err := c.options.FS.Create(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		return c.saveCBZ(downloadedPages, file, comicInfoXML, options.ComicInfoXMLOptions)
	case FormatImages:
		if err := c.options.FS.MkdirAll(path, c.options.ModeDir); err != nil {
			return "", err
		}

		for i, page := range downloadedPages {
			name := fmt.Sprintf("%04d%s", i+1, page.Extension())
			err := afero.WriteFile(
				c.options.FS,
				filepath.Join(path, name),
				page.Image(),
				c.options.ModeFile,
			)
			if err != nil {
				return "", err
			}
		}

		return ciXmlStatusSkip, nil
	default:
		// format validation was done before
		panic("unreachable")
	}
}

// getComicInfoXML gets the ComicInfoXML for the chapter.
//
// It tries to check if chapter implements ChapterWithComicInfoXML
// in case of failure it will use the provided metadata.
//
// The metadata that it uses as fallback could be set by the provider,
// by the client or by libmangal (when no metadata is found it searches for it).
func (c *Client) getComicInfoXML(
	mangaChapter mangadata.Chapter,
	metaChapter metadata.Chapter,
) (metadata.ComicInfoXML, error) {
	withComicInfoXML, ok := mangaChapter.(mangadata.ChapterWithComicInfoXML)
	if ok {
		comicInfo, found, err := withComicInfoXML.ComicInfoXML()
		if err != nil {
			return metadata.ComicInfoXML{}, err
		}
		if found {
			return comicInfo, nil
		}
	}
	return mangaChapter.Volume().Manga().Metadata().ComicInfoXML(metaChapter), nil
}

// savePDF saves pages in FormatPDF
func (c *Client) savePDF(
	pages []mangadata.PageWithImage,
	out io.Writer,
) error {
	c.logger.Log("saving %d pages as PDF", len(pages))

	// convert to readers
	images := make([]io.Reader, len(pages))
	for i, page := range pages {
		images[i] = bytes.NewReader(page.Image())
	}

	return api.ImportImages(nil, out, images, nil, nil)
}

// saveCBZ saves pages in FormatCBZ
func (c *Client) saveCBZ(
	pages []mangadata.PageWithImage,
	out io.Writer,
	comicInfoXml *metadata.ComicInfoXML,
	options metadata.ComicInfoXMLOptions,
) (metadata.DownloadStatus, error) {
	c.logger.Log("saving %d pages as CBZ", len(pages))

	zipWriter := zip.NewWriter(out)
	defer zipWriter.Close()

	for i, page := range pages {
		writer, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:     fmt.Sprintf("%04d%s", i+1, page.Extension()),
			Method:   zip.Store,
			Modified: time.Now(),
		})
		if err != nil {
			return "", err
		}

		_, err = writer.Write(page.Image())
		if err != nil {
			return "", nil
		}
	}

	ciXmlStatus := metadata.DownloadStatusMissingMetadata
	if comicInfoXml != nil {
		ciXmlStatus = metadata.DownloadStatusNew
		marshalled, err := comicInfoXml.Marshal(options)
		if err != nil {
			return "", err
		}

		writer, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:     metadata.FilenameComicInfoXML,
			Method:   zip.Store,
			Modified: time.Now(),
		})
		if err != nil {
			return "", err
		}

		_, err = writer.Write(marshalled)
		if err != nil {
			return "", err
		}
	}

	return ciXmlStatus, nil
}

func (c *Client) saveTAR(
	pages []mangadata.PageWithImage,
	out io.Writer,
) error {
	c.logger.Log("saving %d pages as TAR", len(pages))

	tarWriter := tar.NewWriter(out)
	defer tarWriter.Close()

	for i, page := range pages {
		image := page.Image()
		err := tarWriter.WriteHeader(&tar.Header{
			Name:    fmt.Sprintf("%04d%s", i+1, page.Extension()),
			Size:    int64(len(image)),
			Mode:    int64(c.options.ModeFile),
			ModTime: time.Now(),
		})
		if err != nil {
			return err
		}

		_, err = tarWriter.Write(image)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) saveTARGZ(
	pages []mangadata.PageWithImage,
	out io.Writer,
) error {
	c.logger.Log("bundling TAR into GZIP")

	gzipWriter := gzip.NewWriter(out)
	defer gzipWriter.Close()

	return c.saveTAR(pages, gzipWriter)
}

func (c *Client) saveZIP(
	pages []mangadata.PageWithImage,
	out io.Writer,
) error {
	c.logger.Log("saving %d pages as ZIP", len(pages))

	zipWriter := zip.NewWriter(out)
	defer zipWriter.Close()

	for i, page := range pages {
		writer, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:     fmt.Sprintf("%04d%s", i+1, page.Extension()),
			Method:   zip.Store,
			Modified: time.Now(),
		})
		if err != nil {
			return err
		}

		_, err = writer.Write(page.Image())
		if err != nil {
			return err
		}
	}

	return nil
}

// downloadMangaImage will download image related to manga.
//
// For example this can be either banner image or cover image.
//
// Manga is required to set Referer header.
func (c *Client) downloadMangaImage(
	ctx context.Context,
	manga mangadata.Manga,
	mangaImage mangaImage,
	out io.Writer,
) error {
	c.logger.Log("downloading %s image", mangaImage)
	var URL string
	switch mangaImage {
	case mangaImageCover:
		URL = getCover(manga)
	case mangaImageBanner:
		URL = getBanner(manga)
	default:
		return fmt.Errorf("unknown manga image type %q to download", mangaImage)
	}
	if URL == "" {
		msgFmt := "%s image url not found"
		c.logger.Log(msgFmt, mangaImage)
		return fmt.Errorf(msgFmt, mangaImage)
	}
	c.logger.Log("%s image url: %s", mangaImage, URL)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, nil)
	if err != nil {
		return err
	}

	// TODO: change referer? this asumes that the cover/banner URL comes
	// from the manga site itself, what if it comes from anilist?
	request.Header.Set("Referer", manga.Info().URL)
	request.Header.Set("User-Agent", c.options.UserAgent)
	request.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

	response, err := c.options.HTTPClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http status: %s", response.Status)
	}

	_, err = io.Copy(out, response.Body)
	return err
}

func getCover(manga mangadata.Manga) string {
	cover := manga.Info().Cover
	if cover != "" {
		return cover
	}
	if manga.Metadata() != nil {
		cover = manga.Metadata().CoverImage
	}
	return cover
}

func getBanner(manga mangadata.Manga) string {
	banner := manga.Info().Banner
	if banner != "" {
		return banner
	}
	if manga.Metadata() != nil {
		banner = manga.Metadata().BannerImage
	}
	return banner
}

// getSeriesJSON gets SeriesJSON from the chapter.
//
// It tries to check if chapter manga implements MangaWithSeriesJSON
// in case of failure it will use the provided metadata.
//
// The metadata that it uses as fallback could be set by the provider,
// by the client or by libmangal (when no metadata is found it searches for it).
func (c *Client) getSeriesJSON(
	manga mangadata.Manga,
) (metadata.SeriesJSON, error) {
	withSeriesJSON, ok := manga.(mangadata.MangaWithSeriesJSON)
	if ok {
		seriesJSON, found, err := withSeriesJSON.SeriesJSON()
		if err != nil {
			return metadata.SeriesJSON{}, err
		}
		if found {
			return seriesJSON, nil
		}
	}

	return manga.Metadata().SeriesJSON(), nil
}

func (c *Client) writeSeriesJSON(
	manga mangadata.Manga,
	out io.Writer,
) error {
	c.logger.Log("writing %s", metadata.FilenameSeriesJSON)

	seriesJSON, err := c.getSeriesJSON(manga)
	if err != nil {
		return err
	}

	marshalled, err := seriesJSON.Marshal()
	if err != nil {
		return err
	}

	_, err = out.Write(marshalled)
	return err
}

// TODO: move into anilist? this is specific to anilist read progress
func (c *Client) markChapterAsRead(
	ctx context.Context,
	chapter mangadata.Chapter,
) error {
	chapterMangaInfo := chapter.Volume().Manga().Info()

	var titleToSearch string

	if title := chapterMangaInfo.AnilistSearch; title != "" {
		titleToSearch = title
	} else if title := chapterMangaInfo.Title; title != "" {
		titleToSearch = title
	} else {
		return fmt.Errorf("can't find title for chapter %q", chapter)
	}

	manga, ok, err := c.Anilist().FindClosestManga(ctx, titleToSearch)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("manga for chapter %q was not found on anilist", chapter)
	}

	progress := int(math.Trunc(float64(chapter.Info().Number)))
	return c.Anilist().SetMangaProgress(ctx, manga.ID, progress)
}
