//
// This program creates a gallery website. A gallery is made up of one or
// more albums of images.
//
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/horgh/gallery"
)

// Args holds command line argument information.
type Args struct {
	// Path to a file describing gallery to build.
	GalleryFile string

	// Path to a directory to output the HTML and images.
	InstallDir string

	// Name of the gallery. Human readable.
	Name string

	// Whether to log verbosely.
	Verbose bool

	// Whether to generate/link zips of images.
	IncludeZips bool

	// See description of this option in Album.
	IncludeOriginals bool

	// Force generation of images (e.g. thumbs) even if they exist.
	ForceGenerateImages bool

	// Force generation of HTML even if it exists.
	ForceGenerateHTML bool

	// Force generation of Zips even if they exist.
	ForceGenerateZip bool

	// Images per page (inside albums).
	PageSize int

	// Number of workers to use when resizing images.
	Workers int
}

func main() {
	log.SetFlags(0)

	args, err := getArgs()
	if err != nil {
		log.Printf("Invalid argument: %s", err)
		log.Printf("Usage: %s [arguments]", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	gallery := &gallery.Gallery{
		File:                args.GalleryFile,
		InstallDir:          args.InstallDir,
		Name:                args.Name,
		Verbose:             args.Verbose,
		IncludeZips:         args.IncludeZips,
		IncludeOriginals:    args.IncludeOriginals,
		ForceGenerateImages: args.ForceGenerateImages,
		ForceGenerateHTML:   args.ForceGenerateHTML,
		ForceGenerateZip:    args.ForceGenerateZip,
		PageSize:            args.PageSize,
		Workers:             args.Workers,
	}

	err = gallery.Install()
	if err != nil {
		log.Fatalf("Unable to install gallery: %s", err)
	}
}

func getArgs() (*Args, error) {
	galleryFile := flag.String("gallery-file", "", "Path to a file describing the gallery to build.")
	installDir := flag.String("install-dir", "", "Path to a directory to output HTML/images.")
	title := flag.String("title", "Gallery", "Name/title of the gallery.")
	verbose := flag.Bool("verbose", false, "Toggle verbose logging.")
	includeZips := flag.Bool("include-zips", false, "Generate and link zip files containing images.")
	includeOriginals := flag.Bool("include-originals", true, "Copy original images and link to them from the single image page")
	pageSize := flag.Int("page-size", 50, "Number of image thumbnails per page in albums.")
	forceGenerateImages := flag.Bool("generate-images", false, "Force regenerating resized images. Normally we only do so if they don't exist.")
	forceGenerateHTML := flag.Bool("generate-html", false, "Force regenerating HTML. Normally we only do so if it does not exist.")
	forceGenerateZip := flag.Bool("generate-zip", false, "Force regenerating zip files. Normally we only do so if they do not exist.")
	workers := flag.Int("workers", 4, "Number of workers for image resizing.")

	flag.Parse()

	if len(*galleryFile) == 0 {
		return nil, fmt.Errorf("you must provide a gallery file")
	}

	if len(*installDir) == 0 {
		return nil, fmt.Errorf("you must provide an install directory")
	}

	if len(*title) == 0 {
		return nil, fmt.Errorf("you must provide a title")
	}

	return &Args{
		GalleryFile:         *galleryFile,
		InstallDir:          *installDir,
		Name:                *title,
		Verbose:             *verbose,
		IncludeZips:         *includeZips,
		IncludeOriginals:    *includeOriginals,
		PageSize:            *pageSize,
		ForceGenerateImages: *forceGenerateImages,
		ForceGenerateHTML:   *forceGenerateHTML,
		ForceGenerateZip:    *forceGenerateZip,
		Workers:             *workers,
	}, nil
}
