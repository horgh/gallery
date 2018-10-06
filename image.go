package gallery

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/horgh/magick"
)

// Image holds image information from the metadata file.
type Image struct {
	// Full path to the image.
	Path string

	// Image's base filename.
	Filename string

	// Human readable description of the image.
	Description string

	// Tags assigned to the image.
	Tags []string

	// Size for the thumbnail. Height/width in pixels.
	ThumbnailSize int

	// Size for the larger version of the image (which is still likely smaller
	// than the original image). Maximum of width/height in pixels.
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

// Generate all images from the original, if necessary.
func (i *Image) makeImages(dir string, verbose, forceGenerate bool) error {
	if err := i.makeThumbnail(dir, verbose, forceGenerate); err != nil {
		return err
	}

	return i.makeLargeImage(dir, verbose, forceGenerate)
}

// Create a thumbnail image.
//
// It is thumbnailsize by thumbnailsize. We shrink it down then crop.
func (i *Image) makeThumbnail(dir string, verbose, forceGenerate bool) error {
	resizeFile, err := i.getResizedFilename(dir, i.ThumbnailSize, i.ThumbnailSize)
	if err != nil {
		return err
	}

	if !forceGenerate {
		// If the resized version exists, nothing to do.
		if _, err = os.Stat(resizeFile); err == nil {
			i.ThumbnailPath = resizeFile
			i.ThumbnailFilename = path.Base(resizeFile)
			return nil
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf("stat: %s %s", resizeFile, err)
		}
	}

	if verbose {
		log.Printf("Creating image %s...", resizeFile)
	}

	image, err := magick.NewFromFile(i.Path)
	if err != nil {
		return fmt.Errorf("unable to open image: %s: %s", i.Filename, err)
	}

	if err := image.AutoOrient(); err != nil {
		_ = image.Destroy()
		return fmt.Errorf("unable to auto orient: %s: %s", i.Filename, err)
	}

	// Resize.
	if image.Width() > image.Height() {
		if err := image.Resize(fmt.Sprintf("x%d", i.ThumbnailSize)); err != nil {
			_ = image.Destroy()
			return fmt.Errorf("unable to resize image: %s: %s", i.Filename, err)
		}
	} else {
		if err := image.Resize(fmt.Sprintf("%dx", i.ThumbnailSize)); err != nil {
			_ = image.Destroy()
			return fmt.Errorf("unable to resize image: %s: %s", i.Filename, err)
		}
	}

	// Crop the image. Try to centre depending on which dimension is larger.
	xOffset := 0
	yOffset := 0

	if image.Width() > image.Height() {
		diff := image.Width() - image.Height()
		xOffset = diff / 2
	} else if image.Height() > image.Width() {
		diff := image.Height() - image.Width()
		yOffset = diff / 2
	}

	// ! says to ignore aspect ratio.
	geometry := fmt.Sprintf("%dx%d!+%d+%d", i.ThumbnailSize, i.ThumbnailSize,
		xOffset, yOffset)

	if err := image.Crop(geometry); err != nil {
		_ = image.Destroy()
		return fmt.Errorf("unable to crop: %s: %s", i.Filename, err)
	}

	image.PlusRepage()

	if err := image.ToFile(resizeFile); err != nil {
		_ = image.Destroy()
		return fmt.Errorf("unable to save resized image: %s: %s", resizeFile, err)
	}

	if err := image.Destroy(); err != nil {
		return fmt.Errorf("unable to clean up: %s", err)
	}

	i.ThumbnailPath = resizeFile
	i.ThumbnailFilename = path.Base(resizeFile)

	return nil
}

// Make a large version of the image. It is still shrunken from the original in
// most cases.
func (i *Image) makeLargeImage(dir string, verbose, forceGenerate bool) error {
	resizeFile, err := i.getResizedFilename(dir, i.LargeImageSize, -1)
	if err != nil {
		return err
	}

	if !forceGenerate {
		// If the resized version exists, nothing to do.
		if _, err = os.Stat(resizeFile); err == nil {
			i.LargeImagePath = resizeFile
			i.LargeImageFilename = path.Base(resizeFile)
			return nil
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf("stat: %s %s", resizeFile, err)
		}
	}

	if verbose {
		log.Printf("Creating image %s...", resizeFile)
	}

	image, err := magick.NewFromFile(i.Path)
	if err != nil {
		return fmt.Errorf("unable to open image: %s: %s", i.Filename, err)
	}

	if err := image.AutoOrient(); err != nil {
		_ = image.Destroy()
		return fmt.Errorf("unable to auto orient: %s: %s", i.Filename, err)
	}

	// May not need to resize.
	if image.Width() > i.LargeImageSize || image.Height() > i.LargeImageSize {
		if image.Width() > image.Height() {
			if err := image.Resize(fmt.Sprintf("%dx", i.LargeImageSize)); err != nil {
				_ = image.Destroy()
				return fmt.Errorf("unable to resize image: %s: %s", i.Filename, err)
			}
		} else {
			if err := image.Resize(fmt.Sprintf("x%d", i.LargeImageSize)); err != nil {
				_ = image.Destroy()
				return fmt.Errorf("unable to resize image: %s: %s", i.Filename, err)
			}
		}
	}

	if err := image.ToFile(resizeFile); err != nil {
		_ = image.Destroy()
		return fmt.Errorf("unable to save resized image: %s: %s", resizeFile, err)
	}

	if err := image.Destroy(); err != nil {
		return fmt.Errorf("unable to clean up: %s", err)
	}

	i.LargeImagePath = resizeFile
	i.LargeImageFilename = path.Base(resizeFile)

	return nil
}

// getResizedFilename decides the path to the file with the given width/height.
func (i Image) getResizedFilename(dir string, width,
	height int) (string, error) {

	namePieces := strings.Split(i.Filename, ".")

	if len(namePieces) != 2 {
		return "", fmt.Errorf("unexpected filename format")
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
