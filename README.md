<div align="center">
  <img width="150px" alt="logo depicting a cartoon octopus" src="octopus.png">
  <h1>libmangal</h1>
</div>

> **Warning**
> 
> The API is not stable and may change at any time.

This is an *engine* for downloading, managing and tagging manga with native Anilist integration. A powerful wrapper around anything that implements its `Provider` interface.

Designed to be the backend for applications such as CLI, TUI, web apps, gRPC server, etc.

**Note**: This is a fork of [mangalorg/libmangal](https://github.com/mangalorg/libmangal).

## Features

- Smart caching - only download what you need
- Different export formats
  - PDF - chapters stored a single PDF file
  - CBZ - Comic Book ZIP format
  - TAR - TAR archive
  - ZIP - ZIP archive
  - Images - a plain directory of images
- Monolith - no runtime dependencies
- Generates metadata files
  - `ComicInfo.xml` - The ComicInfo.xml file originates from the ComicRack application, which is not developed anymore. The ComicInfo.xml however is used by a variety of applications
  - `series.json` - A JSON file containing metadata about the series. Originates from [mylar3](https://github.com/mylar3/mylar3)
- Automatically populates missing metadata by querying [Anilist](https://anilist.co)
- Filesystem abstraction - can be used with any filesystem that implements [afero](https://github.com/spf13/afero)
    - Remote filesystems
    - In-memory filesystems
    - etc.
- Highly configurable
    - Define how you want to **name** your files
    - Define how you want to **organize** your files
    - Define how you want to **tag** your files
    - Define how you want to **cache** your files
- Cross-platform - every OS that Go compiles to is supported
    - Windows
    - Linux
    - MacOS
    - WASM
    - etc.

## Install

```bash
go get github.com/luevano/libmangal@latest
```

## Providers

- [luaprovider](https://github.com/luevano/luaprovider) - Generic provider based on Lua scripts.

## Apps using libmangal

- [mangal](https://github.com/luevano/mangal) - Advanced CLI manga downloader. Lua scrapers, export formats, anilist integration, fancy TUI and more.
- [mangalcli](https://github.com/mangalorg/mangalcli) - Advanced Manga Downloader with Anilist integration, metadata generation and Lua extensions.

## Credits

Octopus logo: [Octopus icons created by Freepik - Flaticon](https://www.flaticon.com/free-icons/octopus "octopus icons")
