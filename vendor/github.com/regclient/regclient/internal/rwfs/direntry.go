package rwfs

import (
	"io/fs"
)

type DirEntry struct {
	name string
	mode fs.FileMode
	fi   *FileInfo
}

func NewDE(name string, mode fs.FileMode, fi *FileInfo) *DirEntry {
	return &DirEntry{
		name: name,
		mode: mode,
		fi:   fi,
	}
}

// Name is base name of the file
func (de *DirEntry) Name() string {
	return de.name
}

// Type is file mode bits
func (de *DirEntry) Type() fs.FileMode {
	return de.mode
}

func (de *DirEntry) Info() (fs.FileInfo, error) {
	return de.fi, nil
}

// IsDir abbreviation for Mode().IsDir()
func (de *DirEntry) IsDir() bool {
	return de.mode.IsDir()
}
