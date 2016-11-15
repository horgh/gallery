package gallery

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

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
		return fmt.Errorf("Unable to determine path to file: %s", err)
	}

	// If the file is already present then there is nothing to do.
	_, err = os.Stat(newFilename)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("Problem stat'ing file: %s", err)
	}

	origFilename := fmt.Sprintf("%s%c%s", imageDir, os.PathSeparator, i.Filename)

	log.Printf("Shrinking %s to %d%%...", i.Filename, percent)

	_, err = os.Stat(origFilename)
	if err != nil {
		return fmt.Errorf("Stat failure: %s: %s", i.Filename, err)
	}

	cmd := exec.Command("convert", "-resize", fmt.Sprintf("%d%%", percent),
		origFilename, newFilename)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to run command: %s", err)
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
