package gallery

import (
	"fmt"
	"io"
	"os"
)

// copyFile copies the file!
func copyFile(src string, dest string) error {
	if src == dest {
		return nil
	}

	srcFD, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Unable to open file (read): %s", err)
	}

	destFD, err := os.Create(dest)
	if err != nil {
		_ = srcFD.Close()
		return fmt.Errorf("Unable to open file (write): %s", err)
	}

	_, err = io.Copy(destFD, srcFD)
	if err != nil {
		_ = srcFD.Close()
		_ = destFD.Close()
		return fmt.Errorf("Unable to copy file: %s", err)
	}

	err = srcFD.Close()
	if err != nil {
		return fmt.Errorf("Close: %s: %s", src, err)
	}

	err = destFD.Close()
	if err != nil {
		return fmt.Errorf("Close: %s: %s", dest, err)
	}

	return nil
}

func makeDirIfNotExist(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("%s: %s", dir, err)
		}

		err := os.Mkdir(dir, 0755)
		if err != nil {
			return fmt.Errorf("Mkdir: %s: %s", dir, err)
		}

		return nil
	}

	if !fi.IsDir() {
		return fmt.Errorf("File exists but is not a dir: %s", dir)
	}

	return nil
}
