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
		return fmt.Errorf("unable to open file (read): %s", err)
	}

	destFD, err := os.Create(dest)
	if err != nil {
		_ = srcFD.Close()
		return fmt.Errorf("unable to open file (write): %s", err)
	}

	_, err = io.Copy(destFD, srcFD)
	if err != nil {
		_ = srcFD.Close()
		_ = destFD.Close()
		return fmt.Errorf("unable to copy file: %s", err)
	}

	err = srcFD.Close()
	if err != nil {
		return fmt.Errorf("close: %s: %s", src, err)
	}

	err = destFD.Close()
	if err != nil {
		return fmt.Errorf("close: %s: %s", dest, err)
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
			return fmt.Errorf("mkdir: %s: %s", dir, err)
		}

		return nil
	}

	if !fi.IsDir() {
		return fmt.Errorf("file exists but is not a directory: %s", dir)
	}

	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
