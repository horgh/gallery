//
// This program creates a static photo album website.
//
// You provide it a list of filenames and metadata about each, and where the
// files are located. It generates HTML for a static site, and resizes the
// images to create thumbnails as needed.
//
// This program creates a website with a single photo album.
//
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"summercat.com/gallery"
)

// Args holds the command line arguments.
type Args struct {
	// AlbumFile is the path to a file describing each image. Its filename,
	// descriptive text, and tags if any.
	AlbumFile string

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

	album := gallery.Album{
		Name:         args.Title,
		File:         args.AlbumFile,
		OrigImageDir: args.ImageDir,
		ResizedDir:   args.ResizedImageDir,
		InstallDir:   args.InstallDir,
		Tags:         args.Tags,
		ThumbSize:    args.ThumbSize,
		FullSize:     args.FullSize,
		PageSize:     20,
	}

	err = album.Install()
	if err != nil {
		log.Fatalf("Unable to install album: %s", err)
	}
}

// getArgs retrieves and validates command line arguments.
func getArgs() (Args, error) {
	albumFile := flag.String("album-file", "", "Path to the file describing and listing the images in an album.")
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

	if len(*albumFile) == 0 {
		return Args{}, fmt.Errorf("You must provide an album file.")
	}
	args.AlbumFile = *albumFile

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
