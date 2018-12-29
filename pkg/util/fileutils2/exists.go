package fileutils2

import "os"

func Exists(filepath string) bool {
	_, err := os.Lstat(filepath)
	if err != nil {
		return false
	}
	return true
}

func IsFile(filepath string) bool {
	fi, err := os.Lstat(filepath)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode.IsRegular()
}

func IsDir(filepath string) bool {
	fi, err := os.Lstat(filepath)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode.IsDir()
}
