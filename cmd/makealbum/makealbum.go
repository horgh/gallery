// This program creates a static image album website.
//
// It differs from makegallery in that it creates a website for a single album
// of images, whereas makegallery assumes you have multiple albums.
//
// You provide it a list of filenames and metadata about each, and where the
// files are located. It generates HTML for a static site, and resizes the
// images to create thumbnails as needed.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/horgh/gallery"
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

	// InstallDir is where the selected images and HTML ends up. You probably
	// want to wipe this out each run.
	InstallDir string

	// Verbose controls whether to log more verbose output.
	Verbose bool

	// Title we use for the <title> and header of the page.
	Title string

	// How many images to show per page (thumbnails).
	PageSize int

	// Force generation of images (e.g. thumbs) even if they exist.
	ForceGenerateImages bool

	// Force generation of HTML even if it exists.
	ForceGenerateHTML bool

	// Force generation of Zips even if they exist.
	ForceGenerateZip bool

	// Number of workers to use when resizing images.
	Workers int
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
		Name:                args.Title,
		File:                args.AlbumFile,
		OrigImageDir:        args.ImageDir,
		InstallDir:          args.InstallDir,
		ThumbnailSize:       gallery.ThumbnailSize,
		LargeImageSize:      gallery.LargeImageSize,
		PageSize:            args.PageSize,
		Workers:             args.Workers,
		Verbose:             args.Verbose,
		ForceGenerateImages: args.ForceGenerateImages,
		ForceGenerateHTML:   args.ForceGenerateHTML,
		ForceGenerateZip:    args.ForceGenerateZip,
		Tags:                args.Tags,
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
	installDir := flag.String("install-dir", "", "Path to the directory to install to.")
	verbose := flag.Bool("verbose", false, "Toggle verbose logging.")
	title := flag.String("title", "Album", "Title of the album. We use this for the title element and page header.")
	pageSize := flag.Int("page-size", 50, "Number of image thumbnails per page in albums.")
	forceGenerateImages := flag.Bool("generate-images", false, "Force regenerating resized images. Normally we only do so if they don't exist.")
	forceGenerateHTML := flag.Bool("generate-html", false, "Force regenerating HTML. Normally we only do so if it does not exist.")
	forceGenerateZip := flag.Bool("generate-zip", false, "Force regenerating zip files. Normally we only do so if they do not exist.")
	workers := flag.Int("workers", 4, "Number of workers for image resizing.")

	flag.Parse()

	args := Args{}

	if len(*albumFile) == 0 {
		return Args{}, fmt.Errorf("you must provide an album file")
	}
	args.AlbumFile = *albumFile

	if len(*tagString) > 0 {
		rawTags := strings.Split(*tagString, ",")
		for _, tag := range rawTags {
			args.Tags = append(args.Tags, strings.TrimSpace(tag))
		}
	}

	if len(*imageDir) == 0 {
		return Args{}, fmt.Errorf("you must provide an image directory")
	}
	args.ImageDir = *imageDir

	if len(*installDir) == 0 {
		return Args{}, fmt.Errorf("you must provide an install directory")
	}
	args.InstallDir = *installDir

	args.Verbose = *verbose

	if len(*title) == 0 {
		return Args{}, fmt.Errorf("please provide a title")
	}
	args.Title = *title

	args.PageSize = *pageSize
	args.ForceGenerateImages = *forceGenerateImages
	args.ForceGenerateHTML = *forceGenerateHTML
	args.ForceGenerateZip = *forceGenerateZip
	args.Workers = *workers

	return args, nil
}
