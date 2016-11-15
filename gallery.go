package gallery

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Gallery holds information about a full gallery site which contains 1 or
// more albums of images.
type Gallery struct {
	// File describing the gallery and its albums.
	File string

	// Directory where we output resized images.
	ResizedDir string

	// Directory where we output the finished product including images and
	// HTML.
	InstallDir string

	// Name of the gallery. Its title.
	Name string

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

	return nil
}

// load a gallery's information from a gallery file.
//
// This loads all of the gallery's albums too.
//
// Format of the gallery file: It is made of blocks that look like this:
//
// album-name = Name/title of an album
// album-path = Path to the directory containing the album's images.
// album-file = Path to a file describing the album's images.
// album-tags = Comma separated list of tags to use to decide what images
//              from the album to include. If this is empty then we include all
//              images.
func (g *Gallery) load(file string) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(fh)

	albumName := ""
	albumPath := ""
	albumFile := ""
	albumTags := ""

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) == 0 {
			continue
		}

		pieces := strings.SplitN(text, "=", 1)
		if len(pieces) != 2 {
			_ = fh.Close()
			return fmt.Errorf("Malformed line: %s", text)
		}

		pieces[0] = strings.TrimSpace(pieces[0])
		pieces[1] = strings.TrimSpace(pieces[1])

		if pieces[0] == "album-name" {
			if len(albumName) > 0 {
				err := g.loadAlbum(albumName, albumPath, albumFile, albumTags)
				if err != nil {
					_ = fh.Close()
					return err
				}
			}

			albumName = pieces[1]
			continue
		}

		if pieces[0] == "album-path" {
			albumPath = pieces[1]
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

	err = g.loadAlbum(albumName, albumPath, albumFile, albumTags)
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

func (g *Gallery) loadAlbum(name, path, file, tags string) error {
	if len(name) == 0 {
		return fmt.Errorf("Blank name")
	}

	if len(path) == 0 {
		return fmt.Errorf("No path provided")
	}

	if len(file) == 0 {
		return fmt.Errorf("No file provided")
	}

	album := &Album{
		Name:         name,
		OrigImageDir: path,
		File:         file,
	}

	tagsRaw := strings.Split(tags, ",")
	for _, tag := range tagsRaw {
		tag = strings.TrimSpace(tag)
		if len(tag) == 0 {
			continue
		}

		album.Tags = append(album.Tags, tag)
	}

	err := album.LoadAlbumFile()
	if err != nil {
		return err
	}

	g.albums = append(g.albums, album)

	return nil
}
