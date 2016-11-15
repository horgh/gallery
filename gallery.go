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
	// Directory where we output resized images.
	ResizedDir string

	// Directory where we output the finished product including images and
	// HTML.
	InstallDir string

	// Name of the gallery. Its title.
	Name string

	// Albums in the gallery.
	Albums []*Album
}

// Load a gallery's information from a gallery file.
//
// This loads all of the gallery's albums too.
//
// Format of the gallery file: It is made of blocks that look like this:
//
// album-name = Name/title of an album
// album-file = Path to a file describing the album's images.
// album-tags = Comma separated list of tags to use to decide what images
// from the album to include. If this is empty then we include all images.
func (g *Gallery) Load(file string) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(fh)

	albumName := ""
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
				err := g.loadAlbum(albumName, albumFile, albumTags)
				if err != nil {
					_ = fh.Close()
					return err
				}
			}

			albumName = pieces[1]
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

	err = g.loadAlbum(albumName, albumFile, albumTags)
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

func (g *Gallery) loadAlbum(name, file, tags string) error {
	if len(name) == 0 {
		return fmt.Errorf("Blank name")
	}

	if len(file) == 0 {
		return fmt.Errorf("No file provided")
	}

	album := &Album{
		Name: name,
		File: file,
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

	g.Albums = append(g.Albums, album)

	return nil
}
