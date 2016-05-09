/*
 * gallery is a program to create a standalone photo gallery
 * website. It takes a list of filenames with metadata about
 * each, and a directory of images, and can then generate
 * HTML. It can also create thumbnails.
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// args holds the command line arguments.
type args struct {
	mode       string
	metaFile   string
	tags       []string
	imageDir   string
	thumbsDir  string
	installDir string
}

// image holds image information from the metadata file.
type image struct {
	filename    string
	description string
	tags        []string
}

func (i image) String() string {
	return fmt.Sprintf("Filename: %s Description: %s Tags: %v", i.filename, i.description, i.tags)
}

func (i image) hasTag(tag string) bool {
	for _, myTag := range i.tags {
		if myTag == tag {
			return true
		}
	}

	return false
}

// shrink will resize the image to the given percent
// of the original. It will place the resize in the
// given dir with the suffix _<percent> (before the file
// suffix).
// For thumbnails, 4% looks good for the images I am working
// with. For regular size, 20% looks OK.
func (i image) shrink(percent int, dir string) error {
	namePieces := strings.Split(i.filename, ".")
	if len(namePieces) != 2 {
		return fmt.Errorf("Unexpected filename format")
	}

	newFilename := fmt.Sprintf("%s/%s_%d.%s", dir, namePieces[0], percent, namePieces[1])

	cmd := exec.Command("convert", "-resize", fmt.Sprintf("%d%%", pecent), i.filename, newFilename)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to run command: %s", err.Error())
	}

	return nil
}

func main() {
	log.SetFlags(0)

	myArgs, err := getArgs()
	if err != nil {
		log.Printf("Invalid argument: %s", err.Error())
		log.Printf("Usage: %s <arguments>", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	images, err := parseMetaFile(myArgs.metaFile)
	if err != nil {
		log.Fatalf("Unable to parse metadata file: %s", err.Error())
	}

	log.Printf("Parsed %d images", len(images))
	for _, v := range images {
		log.Printf("Image: %s", v)
	}

	chosenImages, err := chooseImages(myArgs.tags, images)
	if err != nil {
		log.Fatalf("Unable to choose images: %s", err.Error())
	}
	log.Printf("Chose %d images", len(chosenImages))
	for _, v := range chosenImages {
		log.Printf("Image: %s", v)
	}

	// Generate thumbnails for all chosen images
	err = generateImages(myArgs.imageDir, myArgs.thumbsDir, chosenImages)
	if err != nil {
		log.Fatalf("Problem generating images: %s", err.Error())
	}

	// Generate HTML with chosen images
	// Copy images and thumbnails to install directory

	log.Printf("Done!")
}

// getArgs retrieves and validates command line arguments.
func getArgs() (args, error) {
	mode := flag.String("mode", "", "Runtime mode. Possible: generate")
	metaFile := flag.String("meta-file", "", "Path to the file describing and listing the images.")
	tagString := flag.String("tags", "", "Include images with these tag(s) only. Separate by commas. Optional.")
	imageDir := flag.String("image-dir", "", "Path to the directory with all images.")
	thumbsDir := flag.String("thumbs-dir", "", "Path to the directory with thumbnail images. May be empty - we will generate thumbnails on demand.")
	installDir := flag.String("install-dir", "", "Path to the directory to install to.")

	flag.Parse()

	myArgs := args{}

	if *mode != "generate" {
		return args{}, fmt.Errorf("Invalid mode: %s", *mode)
	}
	myArgs.mode = *mode

	if len(*metaFile) == 0 {
		return args{}, fmt.Errorf("You must provide a metadata file.")
	}
	myArgs.metaFile = *metaFile

	if len(*tagString) > 0 {
		rawTags := strings.Split(*tagString, ",")
		for _, tag := range rawTags {
			myArgs.tags = append(myArgs.tags, strings.TrimSpace(tag))
		}
	}

	if len(*imageDir) == 0 {
		return args{}, fmt.Errorf("You must provide an image directory.")
	}
	myArgs.imageDir = *imageDir

	if len(*thumbsDir) == 0 {
		return args{}, fmt.Errorf("You must provide a thumbnails directory.")
	}
	myArgs.thumbsDir = *thumbsDir

	if len(*installDir) == 0 {
		return args{}, fmt.Errorf("You must provide an install directory.")
	}
	myArgs.installDir = *installDir

	return myArgs, nil
}

// parseMetaFile reads in a file listing images.
// Format:
// filename\n
// Description\n
// Optional: Tag: comma separated tags\n
// Blank line
// next filename
func parseMetaFile(filename string) ([]image, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Unable to open: %s: %s", filename, err.Error())
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)

	var images []image

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
			images = append(images, image{
				filename:    imageFilename,
				description: description,
				tags:        tags,
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
		return nil, fmt.Errorf("Scan failure: %s", scanner.Err().Error())
	}

	// May have one last file to store
	if !wantFilename && !wantDescription {
		images = append(images, image{
			filename:    imageFilename,
			description: description,
			tags:        tags,
		})
	}

	return images, nil
}

// chooseImages decides which images we will include when we build
// the HTML.
//
// The basis for this choice is whether the image has one of the requested tags or not.
func chooseImages(tags []string, images []image) ([]image, error) {
	// No tags wanted? Then include everything.
	if len(tags) == 0 {
		return images, nil
	}

	var chosenImages []image

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

// generateImages creates smaller images than the raw ones for
// use in the HTML page.
// This includes one that is "full size" (but still smaller)
// and one that is a thumbnail.
// We place the resized images in the thumbs directory.
// We only resize if the resized image is not already present.
func generateImages(imageDir string, thumbsDir string, images []image) error {
}
