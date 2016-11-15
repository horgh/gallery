package gallery

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Album holds information about an album of images.
type Album struct {
	// Name/title.
	Name string

	// File describing images in the album.
	File string

	// Dir containing the original images.
	OrigImageDir string

	// Dir to output/find resized images in.
	ResizedDir string

	// Dir to install HTML/images.
	InstallDir string

	// Tags tells us to include images that has one of these tags. If there are
	// no tags specified, then include all images.
	Tags []string

	// Image thumb size. Percent.
	ThumbSize int

	// Image "full" size. Percent.
	FullSize int

	// How many images per page.
	PageSize int

	// All available images. Parsed from the album file.
	images []Image

	// A subset of the available images. Those chosen based on tags.
	chosenImages []Image
}

// LoadAlbumFile parses a file listing images and information about them.
//
// Format:
// filename\n
// Description\n
// Optional: Tag: comma separated tags on the image\n
// Blank line
// Then should come the next filename, or end of file.
//
// This means each block describes information about one file.
func (a *Album) LoadAlbumFile() error {
	fh, err := os.Open(a.File)
	if err != nil {
		return fmt.Errorf("Unable to open: %s: %s", a.File, err)
	}

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
				_ = fh.Close()
				return fmt.Errorf("Expecting filename, but have a blank line.")
			}
			wantFilename = false
			wantDescription = true
			continue
		}

		if wantDescription {
			description = scanner.Text()
			if len(description) == 0 {
				_ = fh.Close()
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
			a.images = append(a.images, Image{
				Filename:    imageFilename,
				Description: description,
				Tags:        tags,
			})
			wantFilename = true
			wantDescription = false
			imageFilename = ""
			description = ""
			tags = nil
			continue
		}

		_ = fh.Close()
		return fmt.Errorf("Unexpected line in file: %s", scanner.Text())
	}

	if scanner.Err() != nil {
		_ = fh.Close()
		return fmt.Errorf("Scan failure: %s", scanner.Err())
	}

	// May have one last file to store
	if !wantFilename && !wantDescription {
		a.images = append(a.images, Image{
			Filename:    imageFilename,
			Description: description,
			Tags:        tags,
		})
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
	}

	return nil
}

// Install loads image information, and then chooses, resizes, builds HTML, and
// installs the HTML and images.
func (a *Album) Install() error {
	err := a.LoadAlbumFile()
	if err != nil {
		return fmt.Errorf("Unable to parse metadata file: %s", err)
	}

	err = a.ChooseImages()
	if err != nil {
		return fmt.Errorf("Unable to choose images: %s", err)
	}

	err = a.GenerateImages()
	if err != nil {
		return fmt.Errorf("Problem generating images: %s", err)
	}

	err = a.GenerateHTML()
	if err != nil {
		return fmt.Errorf("Problem generating HTML: %s", err)
	}

	err = a.InstallImages()
	if err != nil {
		return fmt.Errorf("Unable to install images: %s", err)
	}

	return nil
}

// ChooseImages decides which images we will include when we build the HTML.
//
// The basis for this choice is whether the image has one of the requested tags
// or not.
func (a *Album) ChooseImages() error {
	// No tags wanted? Then include everything.
	if len(a.Tags) == 0 {
		a.chosenImages = a.images
		return nil
	}

	for _, image := range a.images {
		for _, wantedTag := range a.Tags {
			if image.hasTag(wantedTag) {
				a.chosenImages = append(a.chosenImages, image)
				break
			}
		}
	}

	return nil
}

// GenerateImages creates smaller images than the raw ones for use in the HTML
// page.
//
// This includes one that is "full size" (but still smaller) and one that is a
// thumbnail. We link to the full size one from the main page. We place the
// resized images in the thumbs directory. We only resize if the resized image
// is not already present. We do this only for chosen images.
func (a *Album) GenerateImages() error {
	err := makeDirIfNotExist(a.ResizedDir)
	if err != nil {
		return err
	}

	for _, image := range a.chosenImages {
		err := image.shrink(a.ThumbSize, a.OrigImageDir, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", a.ThumbSize, err)
		}

		err = image.shrink(a.FullSize, a.OrigImageDir, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", a.FullSize, err)
		}
	}

	return nil
}

// InstallImages copies the chosen images from the resized directory into the
// install directory.
func (a *Album) InstallImages() error {
	err := makeDirIfNotExist(a.InstallDir)
	if err != nil {
		return err
	}

	for _, image := range a.chosenImages {
		thumb, err := image.getResizedFilename(a.ThumbSize, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s", err)
		}

		full, err := image.getResizedFilename(a.FullSize, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full size filename: %s", err)
		}

		thumbTarget := fmt.Sprintf("%s%c%s", a.InstallDir, os.PathSeparator,
			filepath.Base(thumb))

		fullTarget := fmt.Sprintf("%s%c%s", a.InstallDir, os.PathSeparator,
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

// GenerateHTML does just that!
//
// Split over several pages if necessary.
func (a *Album) GenerateHTML() error {
	err := makeDirIfNotExist(a.InstallDir)
	if err != nil {
		return err
	}

	var htmlImages []HTMLImage

	page := 1

	totalPages := len(a.chosenImages) / a.PageSize
	if len(a.chosenImages)%a.PageSize > 0 {
		totalPages++
	}

	for _, image := range a.chosenImages {
		thumbFilename, err := image.getResizedFilename(a.ThumbSize, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s", err)
		}

		fullFilename, err := image.getResizedFilename(a.FullSize, a.ResizedDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full image filename: %s", err)
		}

		htmlImages = append(htmlImages, HTMLImage{
			FullImageURL:  filepath.Base(fullFilename),
			ThumbImageURL: filepath.Base(thumbFilename),
			Description:   image.Description,
		})

		if len(htmlImages) == a.PageSize {
			err := writeHTMLPage(totalPages, len(a.chosenImages), page, htmlImages,
				a.InstallDir, a.Name)
			if err != nil {
				return fmt.Errorf("Unable to generate/write HTML: %s", err)
			}

			htmlImages = nil
			page++
		}
	}

	if len(htmlImages) > 0 {
		err := writeHTMLPage(totalPages, len(a.chosenImages), page, htmlImages,
			a.InstallDir, a.Name)
		if err != nil {
			return fmt.Errorf("Unable to generate/write HTML: %s", err)
		}
	}

	return nil
}
