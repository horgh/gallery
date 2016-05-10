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
	"html/template"
	"io"
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

// hasTag checks if the image has the given tag.
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
func (i image) shrink(percent int, imageDir string, thumbsDir string) error {
	newFilename, err := i.getResizedFilename(percent, thumbsDir)
	if err != nil {
		return fmt.Errorf("Unable to determine path to file: %s", err.Error())
	}

	// If the file is already present then there is nothing to
	// do.
	_, err = os.Stat(newFilename)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("Problem stat'ing file: %s", err.Error())
	}

	origFilename := fmt.Sprintf("%s%c%s", imageDir, os.PathSeparator, i.filename)

	log.Printf("Shrinking %s to %d%%...", i.filename, percent)

	_, err = os.Stat(origFilename)
	if err != nil {
		return fmt.Errorf("Stat failure: %s: %s", i.filename, err.Error())
	}

	cmd := exec.Command("convert", "-resize", fmt.Sprintf("%d%%", percent), origFilename, newFilename)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to run command: %s", err.Error())
	}

	return nil
}

// getResizedFilename gets the filename and path to the file
// with the given shrunk size.
func (i image) getResizedFilename(percent int, thumbsDir string) (string, error) {
	namePieces := strings.Split(i.filename, ".")

	if len(namePieces) != 2 {
		return "", fmt.Errorf("Unexpected filename format")
	}

	newFilename := fmt.Sprintf("%s%c%s_%d.%s", thumbsDir, os.PathSeparator, namePieces[0], percent, namePieces[1])

	return newFilename, nil
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
	pages, err := generateHTML(chosenImages, myArgs.thumbsDir, myArgs.installDir)
	if err != nil {
		log.Fatalf("Problem generating HTML: %s", err.Error())
	}
	log.Printf("Generated %d pages", len(pages))

	// Copy images and thumbnails to install directory
	err = installImages(chosenImages, myArgs.thumbsDir, myArgs.installDir)
	if err != nil {
		log.Fatalf("Unable to install images: %s", err.Error())
	}

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
	for _, image := range images {
		err := image.shrink(4, imageDir, thumbsDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", 4, err.Error())
		}

		err = image.shrink(20, imageDir, thumbsDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", 20, err.Error())
		}
	}

	return nil
}

// generateHTML does just that!
//
// Split over several pages if necessary.
func generateHTML(images []image, thumbsDir string, installDir string) ([]string, error) {
	const tpl = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Gallery</title>
</head>
<body>
{{range .Images}}
<div class="image">
	<a href="{{.Full}}">
		<img src="{{.Thumb}}">
	</a>
	<p>{{.Desc}}</p>
</div><!-- .image -->
{{end}}
</body>
</html>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse HTML template: %s", err.Error())
	}

	type Image struct {
		Full  string
		Thumb string
		Desc  string
	}

	var htmlImages []Image
	for _, img := range images {
		thumb, err := img.getResizedFilename(4, thumbsDir)
		if err != nil {
			return nil, fmt.Errorf("Unable to determine filename: %s", err.Error())
		}

		full, err := img.getResizedFilename(20, thumbsDir)
		if err != nil {
			return nil, fmt.Errorf("Unable to determine filename: %s", err.Error())
		}

		htmlImages = append(htmlImages, Image{
			Full:  basename(full),
			Thumb: basename(thumb),
			Desc:  img.description,
		})
	}

	data := struct {
		Images []Image
	}{
		Images: htmlImages,
	}

	htmlPath := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator, "index.html")

	fh, err := os.Create(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to open HTML file: %s", err.Error())
	}
	defer fh.Close()

	err = t.Execute(fh, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to execute template: %s", err.Error())
	}

	var pages []string

	pages = append(pages, htmlPath)

	return pages, nil
}

// basename determines the name of the file or directory.
// All directory information preceding the lowest will
// be stripped.
func basename(file string) string {
	i := strings.LastIndexByte(file, os.PathSeparator)
	if i == -1 {
		return file
	}

	if i+1 == len(file) {
		return file
	}

	return file[i+1:]
}

// installImages copies the chosen images from the thumbs
// directory into the install directory.
func installImages(images []image, thumbsDir string, installDir string) error {
	for _, image := range images {
		thumb, err := image.getResizedFilename(4, thumbsDir)
		if err != nil {
			return fmt.Errorf("Unable to determine filename: %s", err.Error())
		}

		full, err := image.getResizedFilename(20, thumbsDir)
		if err != nil {
			return fmt.Errorf("Unable to determine filename: %s", err.Error())
		}

		thumbTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator, basename(thumb))

		fullTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator, basename(full))

		err = copyFile(thumb, thumbTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", thumb, thumbTarget, err.Error())
		}

		err = copyFile(full, fullTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", full, fullTarget, err.Error())
		}
	}

	return nil
}

// copyFile copies the file!
func copyFile(src string, dest string) error {
	srcFD, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Unable to open file (read): %s", err.Error())
	}
	defer srcFD.Close()

	destFD, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("Unable to open file (write): %s", err.Error())
	}
	defer destFD.Close()

	_, err = io.Copy(destFD, srcFD)
	if err != nil {
		return fmt.Errorf("Unable to copy file: %s", err.Error())
	}

	return nil
}
