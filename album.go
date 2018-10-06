package gallery

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
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

	// Subdirectory we will be in in the installation directory. We use this when
	// creating the link/image when creating the gallery index page. If you are
	// not building a gallery with multiple albums, then we don't use this field.
	InstallSubDir string

	// The thumbnail size in pixels. Width and height are the same.
	ThumbnailSize int

	// The size of the larger version of images (if the original image is larger
	// than this) in pixels. This is the pixel size set of the longest side.
	LargeImageSize int

	// How many images per page.
	PageSize int

	// Number of workers to use in resizing images.
	Workers int

	// Whether to log verbosely.
	Verbose bool

	// Whether to generate/link zip of images.
	IncludeZip bool

	// If true, we copy over the original images and link to each from the "large"
	// image (the single image pages).
	//
	// If false, we don't, and the large image is not a link.
	IncludeOriginals bool

	// Force generation of images (e.g. thumbs) even if they exist.
	ForceGenerateImages bool

	// Force generation of HTML even if it exists.
	ForceGenerateHTML bool

	// Force generation of Zips even if they exist.
	ForceGenerateZip bool

	// Gallery's name. Human readable.
	//
	// The gallery is the name given to the site holding potentially multiple
	// albums of images. We use it inside the album when linking back to the top
	// level of the gallery. If you are creating only a single album, then we do
	// not use this field.
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
	if err := a.load(); err != nil {
		return fmt.Errorf("unable to parse metadata file: %s", err)
	}

	if err := a.ChooseImages(); err != nil {
		return fmt.Errorf("unable to choose images: %s", err)
	}

	if err := a.GenerateImages(); err != nil {
		return fmt.Errorf("problem generating images: %s", err)
	}

	if err := a.GenerateHTML(); err != nil {
		return fmt.Errorf("problem generating HTML: %s", err)
	}

	if a.IncludeOriginals {
		if err := a.InstallOriginalImages(); err != nil {
			return fmt.Errorf("unable to install original images: %s", err)
		}
	}

	if a.IncludeZip {
		if err := a.makeZip(); err != nil {
			return fmt.Errorf("unable to create zip file: %s", err)
		}
	}

	return nil
}

// ParseAlbumFile an album file. This file lists images and information about
// each of them.
//
// Format of the file:
// Image filename\n
// Optional: Description\n
// Optional: Tag: comma separated tags on the image\n
// Blank line
// Then should come the next filename, or end of file.
//
// This means each block describes information about one file.
//
// We parse into Image structs. We parse only these fields:
// Filename
// Description
// Tags
//
// This is to allow this function to be usable for operating on the album file
// by itself without assuming we are doing anything with it.
func ParseAlbumFile(file string) ([]*Image, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %s: %s", file, err)
	}

	images := []*Image{}

	scanner := bufio.NewScanner(fh)

	filename := ""
	description := ""
	var tags []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) > 0 && line[0] == '#' {
			continue
		}

		if len(filename) == 0 {
			// May have blank lines on their own.
			if len(line) == 0 {
				continue
			}

			filename = line
			continue
		}

		// Blank line ends a block describing one file.
		if len(line) == 0 {
			// May have blank lines on their own.
			if len(filename) == 0 {
				continue
			}

			images = append(images, &Image{
				Filename:    filename,
				Description: description,
				Tags:        tags,
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
		images = append(images, &Image{
			Filename:    filename,
			Description: description,
			Tags:        tags,
		})
	}

	if scanner.Err() != nil {
		_ = fh.Close()
		return nil, fmt.Errorf("scan failure: %s", scanner.Err())
	}

	err = fh.Close()
	if err != nil {
		return nil, fmt.Errorf("close: %s", err)
	}

	return images, nil
}

// load parses an album file to find all of the images, and then fills in
// information about each found Image.
//
// This includes setting each Image's:
// Path
// ThumbnailSize
// LargeImageSize
func (a *Album) load() error {
	images, err := ParseAlbumFile(a.File)
	if err != nil {
		return err
	}

	for _, image := range images {
		image.Path = path.Join(a.OrigImageDir, image.Filename)
		image.ThumbnailSize = a.ThumbnailSize
		image.LargeImageSize = a.LargeImageSize
	}

	a.images = images

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
	if err := makeDirIfNotExist(a.InstallDir); err != nil {
		return err
	}

	ch := make(chan *Image)

	wg := sync.WaitGroup{}

	for i := 0; i < a.Workers; i++ {
		go func(id int) {
			wg.Add(1)
			defer wg.Done()

			for image := range ch {
				err := image.makeImages(a.InstallDir, a.Verbose, a.ForceGenerateImages)
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

// InstallOriginalImages copies the chosen images into the install directory.
func (a *Album) InstallOriginalImages() error {
	for _, image := range a.chosenImages {
		origTarget := path.Join(a.InstallDir, image.Filename)

		// It may be there already.
		if _, err := os.Stat(origTarget); err == nil {
			continue
		}

		err = copyFile(image.Path, origTarget)
		if err != nil {
			return fmt.Errorf("unable to copy %s to %s: %s", image.Path, origTarget,
				err)
		}
	}

	return nil
}

// Make a zip file containing all images in the album.
func (a *Album) makeZip() error {
	zipPath := a.getZipPath()

	// Don't create it if it is there already.
	if !a.ForceGenerateZip {
		_, err := os.Stat(zipPath)
		if err == nil {
			return nil
		}
	}

	if a.Verbose {
		log.Printf("Making zip file: %s...", zipPath)
	}

	zipFH, err := os.Create(zipPath)
	if err != nil {
		return err
	}

	zipWriter := zip.NewWriter(zipFH)

	for _, image := range a.chosenImages {
		imageFH, err := os.Open(image.Path)
		if err != nil {
			_ = zipFH.Close()
			_ = zipWriter.Close()
			return err
		}

		zipFileFH, err := zipWriter.Create(image.Filename)
		if err != nil {
			_ = zipFH.Close()
			_ = zipWriter.Close()
			_ = imageFH.Close()
			return err
		}

		_, err = io.Copy(zipFileFH, imageFH)
		if err != nil {
			_ = zipFH.Close()
			_ = zipWriter.Close()
			_ = imageFH.Close()
			return err
		}

		err = imageFH.Close()
		if err != nil {
			_ = zipFH.Close()
			_ = zipWriter.Close()
			return err
		}
	}

	err = zipWriter.Close()
	if err != nil {
		_ = zipFH.Close()
		return err
	}

	err = zipFH.Close()
	if err != nil {
		return err
	}

	if a.Verbose {
		log.Printf("Wrote zip: %s", zipPath)
	}

	return nil
}

func (a *Album) getZipPath() string {
	return path.Join(a.InstallDir, fmt.Sprintf("%s.zip", a.Name))
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
			IncludeOriginals: a.IncludeOriginals,
			OriginalImageURL: image.Filename,
			ThumbImageURL:    image.ThumbnailFilename,
			FullImageURL:     image.LargeImageFilename,
			Description:      image.Description,
			Index:            i,
		}

		err := makeImagePageHTML(htmlImage, a.InstallDir, len(a.chosenImages),
			a.Name, a.GalleryName, a.Verbose, a.ForceGenerateHTML, page)
		if err != nil {
			return fmt.Errorf("unable to generate image page HTML: %s", err)
		}

		htmlImages = append(htmlImages, htmlImage)

		if len(htmlImages) == a.PageSize {
			err := makeAlbumPageHTML(totalPages, len(a.chosenImages), page,
				htmlImages, a.InstallDir, a.Name, a.GalleryName, a.Verbose,
				a.ForceGenerateHTML, a.IncludeZip)
			if err != nil {
				return fmt.Errorf("unable to generate album page HTML: %s", err)
			}

			htmlImages = nil
			page++
		}
	}

	if len(htmlImages) > 0 {
		err := makeAlbumPageHTML(totalPages, len(a.chosenImages), page, htmlImages,
			a.InstallDir, a.Name, a.GalleryName, a.Verbose, a.ForceGenerateHTML,
			a.IncludeZip)
		if err != nil {
			return fmt.Errorf("unable to generate/write HTML: %s", err)
		}
	}

	return nil
}

// GetThumb picks a thumbnail to represent the album.
func (a *Album) GetThumb() *Image {
	i := rand.Int() % len(a.chosenImages)
	return a.chosenImages[i]
}
