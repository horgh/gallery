package main

import (
	"fmt"
	"io"
	"os"
)

// copyFile copies the file!
func copyFile(src string, dest string) error {
	srcFD, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Unable to open file (read): %s", err)
	}
	defer srcFD.Close()

	destFD, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("Unable to open file (write): %s", err)
	}
	defer destFD.Close()

	_, err = io.Copy(destFD, srcFD)
	if err != nil {
		return fmt.Errorf("Unable to copy file: %s", err)
	}

	return nil
}
