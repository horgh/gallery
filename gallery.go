package gallery

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

// Thumbnails are this size in pixels. Width and height are the same.
const thumbnailSize = 100

// Larger version of images (but smaller than original) have this size in pixels
// as their longest side.
const largeImageSize = 595

// Gallery holds information about a full gallery site which contains 1 or
// more albums of images.
type Gallery struct {
	// File describing the gallery and its albums.
	File string

	// Directory where we output including images and HTML.
	InstallDir string

	// Name of the gallery.
	Name string

	// Whether to log verbosely.
	Verbose bool

	// Force generating images (e.g. thumbs) even if they exist.
	ForceGenerate bool

	// Number of image thumbnails per page in albums.
	PageSize int

	// Number of workers to use in resizing images.
	Workers int

	// Albums in the gallery.
	albums []*Album
}

// Install loads gallery/albums information. It then resizes the images as
// needed, and generates and installs the HTML/images.
func (g *Gallery) Install() error {
	err := g.load(g.File)
	if err != nil {
		return fmt.Errorf("Unable to load gallery file: %s", err)
	}

	err = makeDirIfNotExist(g.InstallDir)
	if err != nil {
		return err
	}

	htmlAlbums := []HTMLAlbum{}

	for _, album := range g.albums {
		err := album.Install()
		if err != nil {
			return fmt.Errorf("Unable to install album: %s: %s", album.Name,
				err)
		}

		htmlAlbums = append(htmlAlbums, HTMLAlbum{
			URL: fmt.Sprintf("%s/index.html", album.InstallSubDir),
			ThumbURL: fmt.Sprintf("%s/%s", album.InstallSubDir,
				album.GetThumb().ThumbnailFilename),
			Name: album.Name,
		})
	}

	err = makeGalleryHTML(g.InstallDir, g.Name, htmlAlbums, g.Verbose)
	if err != nil {
		return fmt.Errorf("Unable to make gallery HTML: %s", err)
	}

	return nil
}

// load a gallery's information from a gallery file.
//
// Format of the gallery file: It is made of blocks that look like this:
//
// album-name   = Name/title of an album. Human readable.
// album-dir    = Path to the directory containing the album's original images.
// album-subdir = A name for the album suitable as a directory name. Not
//                absolute. We install images here and store them here in a
//                subdir to avoid collisions with other albums.
// album-file   = Path to a file describing the album's images.
// album-tags   = Comma separated list of tags to use to decide what images
//                from the album to include. If this is empty then we include
//                all images.
func (g *Gallery) load(file string) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(fh)

	albumName := ""
	albumSubDir := ""
	albumDir := ""
	albumFile := ""
	albumTags := ""

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) == 0 {
			continue
		}

		pieces := strings.SplitN(text, "=", 2)
		if len(pieces) != 2 {
			_ = fh.Close()
			return fmt.Errorf("Malformed line: %s", text)
		}

		pieces[0] = strings.TrimSpace(pieces[0])
		pieces[1] = strings.TrimSpace(pieces[1])

		if pieces[0] == "album-name" {
			if len(albumName) > 0 {
				err := g.loadAlbum(albumName, albumDir, albumSubDir, albumFile,
					albumTags)
				if err != nil {
					_ = fh.Close()
					return err
				}
			}

			albumName = pieces[1]
			continue
		}

		if pieces[0] == "album-dir" {
			albumDir = pieces[1]
			continue
		}

		if pieces[0] == "album-subdir" {
			albumSubDir = pieces[1]
			continue
		}

		if pieces[0] == "album-file" {
			albumFile = pieces[1]
			continue
		}

		if pieces[0] == "album-tags" {
			albumTags = pieces[1]
			continue
		}

		_ = fh.Close()
		return fmt.Errorf("Unexpected line in file: %s", text)
	}

	err = g.loadAlbum(albumName, albumDir, albumSubDir, albumFile, albumTags)
	if err != nil {
		_ = fh.Close()
		return err
	}

	if scanner.Err() != nil {
		return fmt.Errorf("Scanner: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
	}

	return nil
}

func (g *Gallery) loadAlbum(name, dir, subDir, file, tags string) error {
	if len(name) == 0 {
		return fmt.Errorf("Blank name")
	}

	if len(dir) == 0 {
		return fmt.Errorf("No dir provided")
	}

	if len(subDir) == 0 {
		return fmt.Errorf("No subdir provided")
	}

	if len(file) == 0 {
		return fmt.Errorf("No file provided")
	}

	album := &Album{
		Name:           name,
		File:           file,
		OrigImageDir:   dir,
		InstallDir:     path.Join(g.InstallDir, subDir),
		InstallSubDir:  subDir,
		ThumbnailSize:  thumbnailSize,
		LargeImageSize: largeImageSize,
		PageSize:       g.PageSize,
		Workers:        g.Workers,
		Verbose:        g.Verbose,
		ForceGenerate:  g.ForceGenerate,
		GalleryName:    g.Name,
	}

	tagsRaw := strings.Split(tags, ",")
	for _, tag := range tagsRaw {
		tag = strings.TrimSpace(tag)
		if len(tag) == 0 {
			continue
		}

		album.Tags = append(album.Tags, tag)
	}

	g.albums = append(g.albums, album)

	return nil
}
