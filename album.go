package gallery

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
)

// Album holds information about an album of images.
type Album struct {
	// Name.
	Name string

	// File describing images in the album.
	File string

	// Dir containing the original images.
	OrigImageDir string

	// Dir to install HTML/images.
	InstallDir string

	// Subdirectory we will be in in the installation dir.
	InstallSubDir string

	// Image thumbnail size. Width. Pixels.
	ThumbnailSize int

	// Image size. Width. Pixels. This is an image larger than the thumbnail but
	// still likely smaller than the original.
	LargeImageSize int

	// How many images per page.
	PageSize int

	// Number of workers to use in resizing images.
	Workers int

	// Whether to log verbosely.
	Verbose bool

	// Force generating images (e.g. thumbs) even if they exist.
	ForceGenerate bool

	// Gallery's name. Human readable.
	GalleryName string

	// Tags tells us to include images that has one of these tags. If there are
	// no tags specified, then include all images.
	Tags []string

	// All available images. Parsed from the album file.
	images []*Image

	// A subset of the available images. Those chosen based on tags.
	chosenImages []*Image
}

// Install loads image information, and then chooses, resizes, builds HTML, and
// installs the HTML and images.
func (a *Album) Install() error {
	err := a.load()
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

// LoadAlbumFile parses a file listing images and information about them.
//
// Format:
// filename\n
// Optional: Description\n
// Optional: Tag: comma separated tags on the image\n
// Blank line
// Then should come the next filename, or end of file.
//
// This means each block describes information about one file.
func (a *Album) load() error {
	fh, err := os.Open(a.File)
	if err != nil {
		return fmt.Errorf("Unable to open: %s: %s", a.File, err)
	}

	scanner := bufio.NewScanner(fh)

	filename := ""
	description := ""
	var tags []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(filename) == 0 {
			if len(line) == 0 {
				_ = fh.Close()
				return fmt.Errorf("Expecting filename, but have a blank line.")
			}

			filename = line
			continue
		}

		// Blank line ends a block describing one file.
		if len(line) == 0 {
			a.images = append(a.images, &Image{
				Path:           path.Join(a.OrigImageDir, filename),
				Filename:       filename,
				Description:    description,
				Tags:           tags,
				ThumbnailSize:  a.ThumbnailSize,
				LargeImageSize: a.LargeImageSize,
			})

			filename = ""
			description = ""
			tags = nil
			continue
		}

		if strings.HasPrefix(line, "Tag: ") && len(line) > 5 {
			rawTags := strings.Split(line[5:], ",")

			for _, tag := range rawTags {
				tag = strings.TrimSpace(tag)
				if len(tag) == 0 {
					continue
				}

				tags = append(tags, tag)
			}

			continue
		}

		description = line
	}

	// May have one last file to store
	if len(filename) > 0 {
		a.images = append(a.images, &Image{
			Path:           path.Join(a.OrigImageDir, filename),
			Filename:       filename,
			Description:    description,
			Tags:           tags,
			ThumbnailSize:  a.ThumbnailSize,
			LargeImageSize: a.LargeImageSize,
		})
	}

	if scanner.Err() != nil {
		_ = fh.Close()
		return fmt.Errorf("Scan failure: %s", scanner.Err())
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
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

// GenerateImages creates smaller images than the original ones for use in the
// HTML page.
//
// This includes one that is a large size (but still usually smaller than the
// original).
//
// We also generate thumbnails.
//
// We only generate images if the target does not yet exist (unless asked to
// do so).
//
// We only look at chosen images.
func (a *Album) GenerateImages() error {
	err := makeDirIfNotExist(a.InstallDir)
	if err != nil {
		return err
	}

	ch := make(chan *Image)

	wg := sync.WaitGroup{}

	for i := 0; i < a.Workers; i++ {
		go func(id int) {
			wg.Add(1)
			defer wg.Done()

			for image := range ch {
				err := image.makeImages(a.InstallDir, a.Verbose, a.ForceGenerate)
				if err != nil {
					log.Printf("Problem making images: %s", err)
				}
			}
		}(i)
	}

	for _, image := range a.chosenImages {
		ch <- image
	}

	close(ch)

	wg.Wait()

	return nil
}

// InstallImages copies the chosen images into the install directory.
//
// The only images that may not be there yet are the original images.
func (a *Album) InstallImages() error {
	for _, image := range a.chosenImages {
		origTarget := path.Join(a.InstallDir, image.Filename)

		err := copyFile(image.Path, origTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", image.Path, origTarget,
				err)
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

	for i, image := range a.chosenImages {
		htmlImage := HTMLImage{
			OriginalImageURL: image.Filename,
			ThumbImageURL:    image.ThumbnailFilename,
			FullImageURL:     image.LargeImageFilename,
			Description:      image.Description,
			Index:            i,
		}

		htmlImages = append(htmlImages, htmlImage)

		if len(htmlImages) == a.PageSize {
			err := makeAlbumPageHTML(totalPages, len(a.chosenImages), page,
				htmlImages, a.InstallDir, a.Name, a.GalleryName)
			if err != nil {
				return fmt.Errorf("Unable to generate album page HTML: %s", err)
			}

			htmlImages = nil
			page++
		}

		err := makeImagePageHTML(htmlImage, a.InstallDir, len(a.chosenImages),
			a.Name, a.GalleryName)
		if err != nil {
			return fmt.Errorf("Unable to generate image page HTML: %s", err)
		}
	}

	if len(htmlImages) > 0 {
		err := makeAlbumPageHTML(totalPages, len(a.chosenImages), page, htmlImages,
			a.InstallDir, a.Name, a.GalleryName)
		if err != nil {
			return fmt.Errorf("Unable to generate/write HTML: %s", err)
		}
	}

	return nil
}

// GetThumb picks a thumbnail to represent the album.
func (a *Album) GetThumb() *Image {
	i := rand.Int() % len(a.chosenImages)
	return a.chosenImages[i]
}
