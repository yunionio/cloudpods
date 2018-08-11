package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func SavePid(pidfile string) error {
	if err := os.MkdirAll(filepath.Dir(pidfile), os.FileMode(0755)); err != nil {
		return err
	}

	file, err := os.OpenFile(pidfile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error opening pidfile %s: %s", pidfile, err)
	}
	defer file.Close() // in case we fail before the explicit close

	_, err = fmt.Fprintf(file, "%d", os.Getpid())
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}
