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
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
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

// HTMLImage holds image info needed in HTML.
type HTMLImage struct {
	FullImageURL  string
	ThumbImageURL string
	Description   string
}

// Image holds image information from the metadata file.
type Image struct {
	Filename    string
	Description string
	Tags        []string
}

func (i Image) String() string {
	return fmt.Sprintf("Filename: %s Description: %s Tags: %v", i.Filename,
		i.Description, i.Tags)
}

// hasTag checks if the image has the given tag.
func (i Image) hasTag(tag string) bool {
	for _, myTag := range i.Tags {
		if myTag == tag {
			return true
		}
	}

	return false
}

// shrink will resize the image to the given percent of the original.
// It will place the resize in the given dir with the suffix _<percent> (before
// the file suffix).
// For the percentage to use, it really depends on the images you have.
func (i Image) shrink(percent int, imageDir string,
	resizedImageDir string) error {
	newFilename, err := i.getResizedFilename(percent, resizedImageDir)
	if err != nil {
		return fmt.Errorf("Unable to determine path to file: %s", err.Error())
	}

	// If the file is already present then there is nothing to do.
	_, err = os.Stat(newFilename)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("Problem stat'ing file: %s", err.Error())
	}

	origFilename := fmt.Sprintf("%s%c%s", imageDir, os.PathSeparator, i.Filename)

	log.Printf("Shrinking %s to %d%%...", i.Filename, percent)

	_, err = os.Stat(origFilename)
	if err != nil {
		return fmt.Errorf("Stat failure: %s: %s", i.Filename, err.Error())
	}

	cmd := exec.Command("convert", "-resize", fmt.Sprintf("%d%%", percent),
		origFilename, newFilename)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to run command: %s", err.Error())
	}

	return nil
}

// getResizedFilename gets the filename and path to the file with the given
// percentage shrunk size.
func (i Image) getResizedFilename(percent int,
	resizedImageDir string) (string, error) {
	namePieces := strings.Split(i.Filename, ".")

	if len(namePieces) != 2 {
		return "", fmt.Errorf("Unexpected filename format")
	}

	newFilename := fmt.Sprintf("%s%c%s_%d.%s", resizedImageDir, os.PathSeparator,
		namePieces[0], percent, namePieces[1])

	return newFilename, nil
}

func main() {
	log.SetFlags(0)

	args, err := getArgs()
	if err != nil {
		log.Printf("Invalid argument: %s", err.Error())
		log.Printf("Usage: %s <arguments>", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	images, err := parseMetaFile(args.MetaFile)
	if err != nil {
		log.Fatalf("Unable to parse metadata file: %s", err.Error())
	}

	if args.Verbose {
		log.Printf("Parsed %d images", len(images))
		for _, v := range images {
			log.Printf("Image: %s", v)
		}
	}

	chosenImages, err := chooseImages(args.Tags, images)
	if err != nil {
		log.Fatalf("Unable to choose images: %s", err.Error())
	}
	log.Printf("Chose %d images", len(chosenImages))
	for _, v := range chosenImages {
		log.Printf("Image: %s", v)
	}

	// Generate resized images for all chosen images.
	err = generateImages(args.ImageDir, args.ResizedImageDir,
		args.ThumbSize, args.FullSize, chosenImages)
	if err != nil {
		log.Fatalf("Problem generating images: %s", err.Error())
	}

	// Generate HTML with chosen images
	err = generateHTML(chosenImages, args.ResizedImageDir, args.ThumbSize,
		args.FullSize, args.InstallDir, args.Title)
	if err != nil {
		log.Fatalf("Problem generating HTML: %s", err.Error())
	}

	// Copy resized images to the install directory
	err = installImages(chosenImages, args.ResizedImageDir, args.ThumbSize,
		args.FullSize, args.InstallDir)
	if err != nil {
		log.Fatalf("Unable to install images: %s", err.Error())
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
		return nil, fmt.Errorf("Unable to open: %s: %s", filename, err.Error())
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
		return nil, fmt.Errorf("Scan failure: %s", scanner.Err().Error())
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
			return fmt.Errorf("Unable to resize to %d%%: %s", thumbSize, err.Error())
		}

		err = image.shrink(fullSize, imageDir, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to resize to %d%%: %s", fullSize, err.Error())
		}
	}

	return nil
}

// generateHTML does just that!
//
// Split over several pages if necessary.
func generateHTML(images []Image, resizedImageDir string, thumbSize int,
	fullSize int, installDir string, title string) error {
	var htmlImages []HTMLImage

	page := 1

	totalPages := len(images) / pageSize
	if len(images)%pageSize > 0 {
		totalPages++
	}

	for _, img := range images {
		thumbFilename, err := img.getResizedFilename(thumbSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s",
				err.Error())
		}

		fullFilename, err := img.getResizedFilename(fullSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full image filename: %s",
				err.Error())
		}

		htmlImages = append(htmlImages, HTMLImage{
			FullImageURL:  filepath.Base(fullFilename),
			ThumbImageURL: filepath.Base(thumbFilename),
			Description:   img.Description,
		})

		if len(htmlImages) == pageSize {
			err = writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
				title)
			if err != nil {
				return fmt.Errorf("Unable to generate/write HTML: %s", err.Error())
			}

			htmlImages = nil
			page++
		}
	}

	if len(htmlImages) > 0 {
		err := writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
			title)
		if err != nil {
			return fmt.Errorf("Unable to generate/write HTML: %s", err.Error())
		}
	}

	return nil
}

// writeHTMLPage generates and writes an HTML page for the given set of images.
func writeHTMLPage(totalPages int, totalImages int, page int,
	images []HTMLImage, installDir string, title string) error {
	const tpl = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>{{.Title}}</title>
</head>
<body>
<h1>{{.Title}}</h1>
{{range .Images}}
<div class="image">
	<a href="{{.FullImageURL}}">
		<img src="{{.ThumbImageURL}}">
	</a>
	<p>{{.Description}}</p>
</div>
{{end}}
{{if gt .TotalPages 1}}
<p>This is page {{.Page}} of {{.TotalPages}} of images.</p>

{{if gt .Page 1}}
<p><a href="{{.PreviousURL}}">Previous page</a></p>
{{end}}

{{if lt .Page .TotalPages}}
<p><a href="{{.NextURL}}">Next page</a></p>
{{end}}

{{end}}
</body>
</html>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("Unable to parse HTML template: %s", err.Error())
	}

	// Figure out filename to write.
	// Page 1 is index.html. The rest are page-n.html
	filename := "index.html"
	if page > 1 {
		filename = fmt.Sprintf("page-%d.html", page)
	}

	path := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator, filename)

	fh, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Unable to open HTML file: %s", err.Error())
	}
	defer fh.Close()

	previousURL := ""
	if page > 1 {
		if page == 2 {
			previousURL = "index.html"
		} else {
			previousURL = fmt.Sprintf("page-%d.html", page-1)
		}
	}

	nextURL := ""
	if page < totalPages {
		nextURL = fmt.Sprintf("page-%d.html", page+1)
	}

	data := struct {
		Title       string
		Images      []HTMLImage
		TotalPages  int
		Page        int
		PreviousURL string
		NextURL     string
	}{
		Title:       title,
		Images:      images,
		TotalPages:  totalPages,
		Page:        page,
		PreviousURL: previousURL,
		NextURL:     nextURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		return fmt.Errorf("Unable to execute template: %s", err.Error())
	}

	log.Printf("Wrote HTML file: %s", filename)
	return nil
}

// installImages copies the chosen images from the resized directory into the
// install directory.
func installImages(images []Image, resizedImageDir string, thumbSize int,
	fullSize int, installDir string) error {
	for _, image := range images {
		thumb, err := image.getResizedFilename(thumbSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s",
				err.Error())
		}

		full, err := image.getResizedFilename(fullSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full size filename: %s",
				err.Error())
		}

		thumbTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator,
			filepath.Base(thumb))

		fullTarget := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator,
			filepath.Base(full))

		err = copyFile(thumb, thumbTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", thumb, thumbTarget,
				err.Error())
		}

		err = copyFile(full, fullTarget)
		if err != nil {
			return fmt.Errorf("Unable to copy %s to %s: %s", full, fullTarget,
				err.Error())
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
