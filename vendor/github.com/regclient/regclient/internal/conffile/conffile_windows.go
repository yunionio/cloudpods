//go:build windows
// +build windows

package conffile

import (
	"io/fs"
)

const homeEnv = "USERPROFILE"

func getFileOwner(stat fs.FileInfo) (int, int, error) {
	return 0, 0, nil
}
