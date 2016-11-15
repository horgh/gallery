package gallery

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/quirkey/magick"
)

// Image holds image information from the metadata file.
type Image struct {
	// Full path to the iamge.
	Path string

	// Image's basename.
	Filename string

	// Human readable description of the file.
	Description string

	// Tags assigned to the image.
	Tags []string

	// Size for the thumbnail. Width in pixels.
	ThumbnailSize int

	// Size for the larger version of the image (which is still likely smaller
	// than the original image). Width in pixels.
	LargeImageSize int

	// Path to the thumbnail.
	ThumbnailPath string

	// Basename of the thumbnail.
	ThumbnailFilename string

	// Path to the larger version of the image.
	LargeImagePath string

	// Basename of the larger version of the image.
	LargeImageFilename string
}

func (i Image) String() string {
	return fmt.Sprintf("Filename: %s Description: %s Tags: %v", i.Filename,
		i.Description, i.Tags)
}

// hasTag checks if the image has the given tag.
func (i Image) hasTag(tag string) bool {
	for _, myTag := range i.Tags {
		if myTag == tag {
			return true
		}
	}

	return false
}

func (i *Image) makeImages(resizeDir string, verbose bool) error {
	err := i.makeThumbnail(resizeDir, verbose)
	if err != nil {
		return err
	}

	return i.makeLargeImage(resizeDir, verbose)
}

func (i *Image) makeThumbnail(resizeDir string, verbose bool) error {
	imagePath, err := i.resize(resizeDir, i.ThumbnailSize, verbose)
	if err != nil {
		return err
	}

	i.ThumbnailPath = imagePath
	i.ThumbnailFilename = path.Base(imagePath)

	return nil
}

func (i *Image) makeLargeImage(resizeDir string, verbose bool) error {
	imagePath, err := i.resize(resizeDir, i.LargeImageSize, verbose)
	if err != nil {
		return err
	}

	i.LargeImagePath = imagePath
	i.LargeImageFilename = path.Base(imagePath)

	return nil
}

func (i Image) resize(resizeDir string, width int,
	verbose bool) (string, error) {

	resizeFile, err := i.getResizedFilename(resizeDir, width)
	if err != nil {
		return "", err
	}

	// If the resized version exists, nothing to do.
	_, err = os.Stat(resizeFile)
	if err == nil {
		return resizeFile, nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("Stat: %s %s", resizeFile, err)
	}

	if verbose {
		log.Printf("Creating image %s...", resizeFile)
	}

	image, err := magick.NewFromFile(i.Path)
	if err != nil {
		return "", fmt.Errorf("Unable to open image: %s: %s", i.Filename, err)
	}

	err = image.Resize(fmt.Sprintf("%dx", width))
	if err != nil {
		_ = image.Destroy()
		return "", fmt.Errorf("Unable to resize image: %s: %s", i.Filename, err)
	}

	err = image.ToFile(resizeFile)
	if err != nil {
		_ = image.Destroy()
		return "", fmt.Errorf("Unable to save resized image: %s: %s", resizeFile,
			err)
	}

	err = image.Destroy()
	if err != nil {
		return "", fmt.Errorf("Unable to clean up: %s", err)
	}

	return resizeFile, nil
}

// getResizedFilename gets the filename and path to the file with the given
// width.
func (i Image) getResizedFilename(dir string, width int) (string, error) {
	namePieces := strings.Split(i.Filename, ".")

	if len(namePieces) != 2 {
		return "", fmt.Errorf("Unexpected filename format")
	}

	newName := fmt.Sprintf("%s_%d.%s", namePieces[0], width, namePieces[1])

	return path.Join(dir, newName), nil
}
