//go:build plan9 || windows || android
// +build plan9 windows android

package sftp

import (
	"os"
)

func fileStatFromInfoOs(fi os.FileInfo, flags *uint32, fileStat *FileStat) {
	// todo
}