//
// gallery is a program to create a static photo gallery website.
//
// You provide it a list of filenames and metadata about each, and where the
// files are located. It generates HTML for a static site, and resizes the
// images to create thumbnails as needed.
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// pageSize defines how many images to have per page.
const pageSize = 20

// Args holds the command line arguments.
type Args struct {
	// MetaFile is the path to a file describing each image. Its filename,
	// descriptive text, and tags if any.
	MetaFile string

	// Tags, which may be empty, holds the tags of images to include in the
	// build.
	Tags []string

	// ImageDir is where the raw images are found.
	ImageDir string

	// ResizedImageDir is where we place resized images from imageDir.
	// You probably will want to keep that around persistently rather than
	// resizing repeatedly.
	ResizedImageDir string

	// InstallDir is where the selected images and HTML ends up. You probably
	// want to wipe this out each run.
	InstallDir string

	// ThumbSize is the percentage size a thumbnail is of the original.
	// We will resize the original to this percentage.
	ThumbSize int

	// FullSize is the percentage size a full image is of the original.
	// We will resize the original to this percentage.
	FullSize int

	// Verbose controls whether to log more verbose output.
	Verbose bool

	// Title we use for the <title> and header of the page.
	Title string
}

func main() {
	log.SetFlags(0)

	args, err := getArgs()
	if err != nil {
		log.Printf("Invalid argument: %s", err)
		log.Printf("Usage: %s <arguments>", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	images, err := parseMetaFile(args.MetaFile)
	if err != nil {
		log.Fatalf("Unable to parse metadata file: %s", err)
	}

	if args.Verbose {
		log.Printf("Parsed %d images", len(images))
		for _, v := range images {
			log.Printf("Image: %s", v)
		}
	}

	chosenImages, err := chooseImages(args.Tags, images)
	if err != nil {
		log.Fatalf("Unable to choose images: %s", err)
	}
	log.Printf("Chose %d images", len(chosenImages))
	for _, v := range chosenImages {
		log.Printf("Image: %s", v)
	}

	// Generate resized images for all chosen images.
	err = generateImages(args.ImageDir, args.ResizedImageDir,
		args.ThumbSize, args.FullSize, chosenImages)
	if err != nil {
		log.Fatalf("Problem generating images: %s", err)
	}

	// Generate HTML with chosen images
	err = generateHTML(chosenImages, args.ResizedImageDir, args.ThumbSize,
		args.FullSize, args.InstallDir, args.Title)
	if err != nil {
		log.Fatalf("Problem generating HTML: %s", err)
	}

	// Copy resized images to the install directory
	err = installImages(chosenImages, args.ResizedImageDir, args.ThumbSize,
		args.FullSize, args.InstallDir)
	if err != nil {
		log.Fatalf("Unable to install images: %s", err)
	}

	log.Printf("Done!")
}

// getArgs retrieves and validates command line arguments.
func getArgs() (Args, error) {
	metaFile := flag.String("meta-file", "", "Path to the file describing and listing the images.")
	tagString := flag.String("tags", "", "Include images with these tag(s) only. Separate by commas. Optional.")
	imageDir := flag.String("image-dir", "", "Path to the directory with all images.")
	resizedImageDir := flag.String("resized-dir", "", "Path to the directory to hold resized images. We resize on demand.")
	installDir := flag.String("install-dir", "", "Path to the directory to install to.")
	thumbSize := flag.Int("thumb-size", 4, "Resize images to this percent of the original to create thumbnails.")
	fullSize := flag.Int("full-size", 20, "Resize images to this percent of the original to create the 'full' image (linked to by the thumbnail).")
	verbose := flag.Bool("verbose", false, "Toggle more verbose output.")
	title := flag.String("title", "Gallery", "Title of the gallery. We use this for the title element and page header.")

	flag.Parse()

	args := Args{}

	if len(*metaFile) == 0 {
		return Args{}, fmt.Errorf("You must provide a metadata file.")
	}
	args.MetaFile = *metaFile

	if len(*tagString) > 0 {
		rawTags := strings.Split(*tagString, ",")
		for _, tag := range rawTags {
			args.Tags = append(args.Tags, strings.TrimSpace(tag))
		}
	}

	if len(*imageDir) == 0 {
		return Args{}, fmt.Errorf("You must provide an image directory.")
	}
	args.ImageDir = *imageDir

	if len(*resizedImageDir) == 0 {
		return Args{}, fmt.Errorf("You must provide a resized image directory.")
	}
	args.ResizedImageDir = *resizedImageDir

	if len(*installDir) == 0 {
		return Args{}, fmt.Errorf("You must provide an install directory.")
	}
	args.InstallDir = *installDir

	if *thumbSize <= 0 || *thumbSize >= 100 {
		return Args{}, fmt.Errorf("Thumbnail size must be (0, 100).")
	}
	args.ThumbSize = *thumbSize

	if *fullSize <= 0 || *fullSize >= 100 {
		return Args{}, fmt.Errorf("Full image size must be (0, 100).")
	}
	args.FullSize = *fullSize

	args.Verbose = *verbose

	if len(*title) == 0 {
		return Args{}, fmt.Errorf("Please provide a title.")
	}
	args.Title = *title

	return args, nil
}

// parseMetaFile reads in a file listing images and parses it into memory.
// Format:
// filename\n
// Description\n
// Optional: Tag: comma separated tags\n
// Blank line
// Then should come the next filename, or end of file.
func parseMetaFile(filename string) ([]Image, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Unable to open: %s: %s", filename, err)
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)

	var images []Image

	wantFilename := true
	wantDescription := false
	imageFilename := ""
	description := ""
	var tags []string

	for scanner.Scan() {
		if wantFilename {
			imageFilename = scanner.Text()
			if len(imageFilename) == 0 {
				return nil, fmt.Errorf("Expecting filename, but have a blank line.")
			}
			wantFilename = false
			wantDescription = true
			continue
		}

		if wantDescription {
			description = scanner.Text()
			if len(description) == 0 {
				return nil, fmt.Errorf("Expecting description, but have a blank line.")
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
			images = append(images, Image{
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

		return nil, fmt.Errorf("Unexpected line in file: %s", scanner.Text())
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("Scan failure: %s", scanner.Err())
	}

	// May have one last file to store
	if !wantFilename && !wantDescription {
		images = append(images, Image{
			Filename:    imageFilename,
			Description: description,
			Tags:        tags,
		})
	}

	return images, nil
}

// chooseImages decides which images we will include when we build the HTML.
//
// The basis for this choice is whether the image has one of the requested tags
// or not.
func chooseImages(tags []string, images []Image) ([]Image, error) {
	// No tags wanted? Then include everything.
	if len(tags) == 0 {
		return images, nil
	}

	var chosenImages []Image

	for _, image := range images {
		for _, wantedTag := range tags {
			if image.hasTag(wantedTag) {
				chosenImages = append(chosenImages, image)
				break
			}
		}
	}

	return chosenImages, nil
}

// generateImages creates smaller images than the raw ones for use in the HTML
// page.
// This includes one that is "full size" (but still smaller) and one that is a
// thumbnail. We link to the full size one from the main page.
// We place the resized images in the thumbs directory.
// We only resize if the resized image is not already present.
func generateImages(imageDir string, resizedImageDir string, thumbSize int,
	fullSize int, images []Image) error {
	for _, image := range images {
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
func installImages(images []Image, resizedImageDir string, thumbSize int,
	fullSize int, installDir string) error {
	for _, image := range images {
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
