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
	resizeFile, err := i.getResizedFilename(resizeDir, i.ThumbnailSize,
		i.ThumbnailSize)
	if err != nil {
		return err
	}

	// If the resized version exists, nothing to do.
	_, err = os.Stat(resizeFile)
	if err == nil {
		i.ThumbnailPath = resizeFile
		i.ThumbnailFilename = path.Base(resizeFile)
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("Stat: %s %s", resizeFile, err)
	}

	if verbose {
		log.Printf("Creating image %s...", resizeFile)
	}

	image, err := magick.NewFromFile(i.Path)
	if err != nil {
		return fmt.Errorf("Unable to open image: %s: %s", i.Filename, err)
	}

	if image.Width() > image.Height() {
		err := image.Resize(fmt.Sprintf("x%d", i.ThumbnailSize))
		if err != nil {
			_ = image.Destroy()
			return fmt.Errorf("Unable to resize image: %s: %s", i.Filename, err)
		}
	} else {
		err := image.Resize(fmt.Sprintf("%dx", i.ThumbnailSize))
		if err != nil {
			_ = image.Destroy()
			return fmt.Errorf("Unable to resize image: %s: %s", i.Filename, err)
		}
	}

	// Crop the image. Try to centre depending on which dimension is larger.
	xOffset := 0
	yOffset := 0

	if image.Width() > image.Height() {
		diff := image.Width() - image.Height()
		xOffset = diff / 2
	} else if image.Height() > image.Height() {
		diff := image.Height() - image.Width()
		yOffset = diff / 2
	}

	// ! says to ignore aspect ratio.
	geometry := fmt.Sprintf("%dx%d!+%d+%d", i.ThumbnailSize, i.ThumbnailSize,
		xOffset, yOffset)

	err = image.Crop(geometry)
	if err != nil {
		_ = image.Destroy()
		return fmt.Errorf("Unable to crop: %s: %s", i.Filename, err)
	}

	err = image.ToFile(resizeFile)
	if err != nil {
		_ = image.Destroy()
		return fmt.Errorf("Unable to save resized image: %s: %s", resizeFile, err)
	}

	err = image.Destroy()
	if err != nil {
		return fmt.Errorf("Unable to clean up: %s", err)
	}

	i.ThumbnailPath = resizeFile
	i.ThumbnailFilename = path.Base(resizeFile)

	return nil
}

func (i *Image) makeLargeImage(resizeDir string, verbose bool) error {
	resizeFile, err := i.getResizedFilename(resizeDir, i.LargeImageSize, -1)
	if err != nil {
		return err
	}

	// If the resized version exists, nothing to do.
	_, err = os.Stat(resizeFile)
	if err == nil {
		i.LargeImagePath = resizeFile
		i.LargeImageFilename = path.Base(resizeFile)
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("Stat: %s %s", resizeFile, err)
	}

	if verbose {
		log.Printf("Creating image %s...", resizeFile)
	}

	image, err := magick.NewFromFile(i.Path)
	if err != nil {
		return fmt.Errorf("Unable to open image: %s: %s", i.Filename, err)
	}

	if image.Width() > image.Height() {
		err := image.Resize(fmt.Sprintf("%dx", i.LargeImageSize))
		if err != nil {
			_ = image.Destroy()
			return fmt.Errorf("Unable to resize image: %s: %s", i.Filename, err)
		}
	} else {
		err := image.Resize(fmt.Sprintf("x%d", i.LargeImageSize))
		if err != nil {
			_ = image.Destroy()
			return fmt.Errorf("Unable to resize image: %s: %s", i.Filename, err)
		}
	}

	err = image.ToFile(resizeFile)
	if err != nil {
		_ = image.Destroy()
		return fmt.Errorf("Unable to save resized image: %s: %s", resizeFile, err)
	}

	err = image.Destroy()
	if err != nil {
		return fmt.Errorf("Unable to clean up: %s", err)
	}

	i.LargeImagePath = resizeFile
	i.LargeImageFilename = path.Base(resizeFile)

	return nil
}

// getResizedFilename gets the filename and path to the file with the given
// width.
func (i Image) getResizedFilename(dir string, width, height int) (string, error) {
	namePieces := strings.Split(i.Filename, ".")

	if len(namePieces) != 2 {
		return "", fmt.Errorf("Unexpected filename format")
	}

	// -1 if the width/height is auto. Width/height will be width depending on
	// which is larger.
	newName := ""
	if height != -1 {
		newName = fmt.Sprintf("%s_%d_%d.%s", namePieces[0], width, height,
			namePieces[1])
	} else {
		newName = fmt.Sprintf("%s_%d.%s", namePieces[0], width, namePieces[1])
	}

	return path.Join(dir, newName), nil
}
