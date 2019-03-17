package utils

import (
	"os"
)

const (
	FileModeDir           = os.FileMode(0755)
	FileModeFile          = os.FileMode(0644)
	FileModeFileExec      = os.FileMode(0755)
	FileModeDirSensitive  = os.FileMode(0700)
	FileModeFileSensitive = os.FileMode(0600)
)
