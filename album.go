package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Album holds information about an album of images.
type Album struct {
	Images       []Image
	ChosenImages []Image
}

// parseMetaFile reads in a file listing images and parses it into memory.
// Format:
// filename\n
// Description\n
// Optional: Tag: comma separated tags\n
// Blank line
// Then should come the next filename, or end of file.
func (a *Album) parseMetaFile(filename string) error {
	fh, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Unable to open: %s: %s", filename, err)
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)

	wantFilename := true
	wantDescription := false
	imageFilename := ""
	description := ""
	var tags []string

	for scanner.Scan() {
		if wantFilename {
			imageFilename = scanner.Text()
			if len(imageFilename) == 0 {
				return fmt.Errorf("Expecting filename, but have a blank line.")
			}
			wantFilename = false
			wantDescription = true
			continue
		}

		if wantDescription {
			description = scanner.Text()
			if len(description) == 0 {
				return fmt.Errorf("Expecting description, but have a blank line.")
			}
			wantDescription = false
			continue
		}

		// May have Tag line, or a blank line.

		if strings.HasPrefix(scanner.Text(), "Tag: ") &&
			len(scanner.Text()) > 5 {
			rawTags := strings.Split(scanner.Text()[5:], ",")
			for _, tag := range rawTags {
				tags = append(tags, strings.TrimSpace(tag))
			}
			continue
		}

		if len(scanner.Text()) == 0 {
			a.Images = append(a.Images, Image{
				Filename:    imageFilename,
				Description: description,
				Tags:        tags,
			})
			wantFilename = true
			wantDescription = false
			filename = ""
			description = ""
			tags = nil
			continue
		}

		return fmt.Errorf("Unexpected line in file: %s", scanner.Text())
	}

	if scanner.Err() != nil {
		return fmt.Errorf("Scan failure: %s", scanner.Err())
	}

	// May have one last file to store
	if !wantFilename && !wantDescription {
		a.Images = append(a.Images, Image{
			Filename:    imageFilename,
			Description: description,
			Tags:        tags,
		})
	}

	return nil
}

// chooseImages decides which images we will include when we build the HTML.
//
// The basis for this choice is whether the image has one of the requested tags
// or not.
func (a *Album) chooseImages(tags []string) error {
	// No tags wanted? Then include everything.
	if len(tags) == 0 {
		a.ChosenImages = a.Images
		return nil
	}

	for _, image := range a.Images {
		for _, wantedTag := range tags {
			if image.hasTag(wantedTag) {
				a.ChosenImages = append(a.ChosenImages, image)
				break
			}
		}
	}

	return nil
}

// generateImages creates smaller images than the raw ones for use in the HTML
// page.
// This includes one that is "full size" (but still smaller) and one that is a
// thumbnail. We link to the full size one from the main page.
// We place the resized images in the thumbs directory.
// We only resize if the resized image is not already present.
// We do this only for chosen images.
func (a *Album) generateImages(imageDir string, resizedImageDir string,
	thumbSize int, fullSize int) error {
	for _, image := range a.ChosenImages {
		err := image.shrink(thumbSize, imageDir, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", thumbSize, err)
		}

		err = image.shrink(fullSize, imageDir, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", fullSize, err)
		}
	}

	return nil
}

// installImages copies the chosen images from the resized directory into the
// install directory.
func (a *Album) installImages(resizedImageDir string, thumbSize int,
	fullSize int, installDir string) error {
	for _, image := range a.ChosenImages {
		thumb, err := image.getResizedFilename(thumbSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s", err)
		}

		full, err := image.getResizedFilename(fullSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full size filename: %s", err)
		}

		thumbTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator,
			filepath.Base(thumb))

		fullTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator,
			filepath.Base(full))

		err = copyFile(thumb, thumbTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", thumb, thumbTarget, err)
		}

		err = copyFile(full, fullTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", full, fullTarget, err)
		}
	}

	return nil
}

// generateHTML does just that!
//
// Split over several pages if necessary.
func (a *Album) generateHTML(resizedImageDir string, thumbSize int,
	fullSize int, installDir string, title string) error {
	var htmlImages []HTMLImage

	page := 1

	totalPages := len(a.ChosenImages) / pageSize
	if len(a.ChosenImages)%pageSize > 0 {
		totalPages++
	}

	for _, image := range a.ChosenImages {
		thumbFilename, err := image.getResizedFilename(thumbSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s", err)
		}

		fullFilename, err := image.getResizedFilename(fullSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full image filename: %s", err)
		}

		htmlImages = append(htmlImages, HTMLImage{
			FullImageURL:  filepath.Base(fullFilename),
			ThumbImageURL: filepath.Base(thumbFilename),
			Description:   image.Description,
		})

		if len(htmlImages) == pageSize {
			err = writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
				title)
			if err != nil {
				return fmt.Errorf("Unable to generate/write HTML: %s", err)
			}

			htmlImages = nil
			page++
		}
	}

	if len(htmlImages) > 0 {
		err := writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
			title)
		if err != nil {
			return fmt.Errorf("Unable to generate/write HTML: %s", err)
		}
	}

	return nil
}
