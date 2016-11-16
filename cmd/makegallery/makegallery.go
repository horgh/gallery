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

	"summercat.com/gallery"
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

	// Force generating images (e.g. thumbs) even if they exist.
	ForceGenerate bool

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
		File:          args.GalleryFile,
		InstallDir:    args.InstallDir,
		Name:          args.Name,
		Verbose:       args.Verbose,
		ForceGenerate: args.ForceGenerate,
		PageSize:      args.PageSize,
		Workers:       args.Workers,
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
	pageSize := flag.Int("page-size", 50, "Number of image thumbnails per page in albums.")
	forceGenerate := flag.Bool("force-generate", false, "Force regenerating resized images. Normally we only do so if they don't exist.")
	workers := flag.Int("workers", 4, "Number of workers for image resizing.")

	flag.Parse()

	if len(*galleryFile) == 0 {
		return nil, fmt.Errorf("You must provide a gallery file.")
	}

	if len(*installDir) == 0 {
		return nil, fmt.Errorf("You must provide an install directory.")
	}

	if len(*title) == 0 {
		return nil, fmt.Errorf("You must provide a title.")
	}

	return &Args{
		GalleryFile:   *galleryFile,
		InstallDir:    *installDir,
		Name:          *title,
		Verbose:       *verbose,
		PageSize:      *pageSize,
		ForceGenerate: *forceGenerate,
		Workers:       *workers,
	}, nil
}
