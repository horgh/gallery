// This program helps you add images to an existing album.
//
// It is possible that you created an album and described images in its album
// description file, and then later you find more images to add. If there are a
// lot of images, then adding the new ones in the correct order in the file is
// tedious. This program adds the new images into the file in the correct spot
// by merging two album files.
//
// Note in order for this program to work the images must be named in a
// sortable way. For example, IMG_20170213, IMG_20170214, etc. The merging
// occurs based on sorting by the image's filename.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/horgh/gallery"
)

// Args hold command line arguments.
type Args struct {
	OriginalAlbumFile string
	NewAlbumFile      string
	OutputFile        string
}

// ByFilename is a type for sorting.
type ByFilename []*gallery.Image

func (f ByFilename) Len() int           { return len(f) }
func (f ByFilename) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f ByFilename) Less(i, j int) bool { return f[i].Filename < f[j].Filename }

func main() {
	args, err := getArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.PrintDefaults()
		os.Exit(1)
	}

	origImages, err := gallery.ParseAlbumFile(args.OriginalAlbumFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse album file: %s: %s",
			args.OriginalAlbumFile, err)
		os.Exit(1)
	}

	newImages, err := gallery.ParseAlbumFile(args.NewAlbumFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse album file: %s: %s",
			args.NewAlbumFile, err)
		os.Exit(1)
	}

	allImages := []*gallery.Image{}
	allImages = append(allImages, origImages...)
	allImages = append(allImages, newImages...)

	sort.Sort(ByFilename(allImages))

	err = writeAlbumFile(args.OutputFile, allImages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to write album file: %s: %s",
			args.OutputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Wrote album file: %s\n", args.OutputFile)
}

func getArgs() (*Args, error) {
	origAlbumFile := flag.String("album-file", "", "Path to an existing album file.")
	newAlbumFile := flag.String("new-album-file", "", "Path to the album file with the new images.")
	outputFile := flag.String("output-file", "", "Path to the new album file to write.")

	flag.Parse()

	if len(*origAlbumFile) == 0 {
		return nil, fmt.Errorf("you must provide the original album file")
	}

	if len(*newAlbumFile) == 0 {
		return nil,
			fmt.Errorf("you must provide the album file with the new images")
	}

	if len(*outputFile) == 0 {
		return nil, fmt.Errorf("you must provide an output file")
	}

	return &Args{
		OriginalAlbumFile: *origAlbumFile,
		NewAlbumFile:      *newAlbumFile,
		OutputFile:        *outputFile,
	}, nil
}

func readImagesList(file string) ([]string, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := fh.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "close: %s: %s", file, err)
		}
	}()

	images := []string{}
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		images = append(images, line)
	}

	return images, nil
}

// For the format of the file we write, refer to gallery.ParseAlbumFile()
func writeAlbumFile(file string, images []*gallery.Image) error {
	fh, err := os.Create(file)
	if err != nil {
		return err
	}

	defer func() {
		err := fh.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Problem closing file: %s: %s", file, err)
		}
	}()

	for _, image := range images {
		err := write(fh, image.Filename+"\n")
		if err != nil {
			return err
		}

		if len(image.Description) > 0 {
			err := write(fh, image.Description+"\n")
			if err != nil {
				return err
			}
		}

		if len(image.Tags) > 0 {
			tagStr := strings.Join(image.Tags, ", ")
			err := write(fh, tagStr+"\n")
			if err != nil {
				return err
			}
		}

		err = write(fh, "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func write(fh *os.File, s string) error {
	n, err := fh.WriteString(s)
	if err != nil {
		return fmt.Errorf("unable to write: %s", err)
	}

	if n != len(s) {
		return fmt.Errorf("short write: wrote %d, wanted to write %d", n, len(s))
	}

	return nil
}
