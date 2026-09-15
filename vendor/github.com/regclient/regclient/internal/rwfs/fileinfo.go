package rwfs

import (
	"io/fs"
	"time"
)

type FileInfo struct {
	name string
	size int64
	mod  time.Time
	mode fs.FileMode
}

func NewFI(name string, size int64, mod time.Time, mode fs.FileMode) *FileInfo {
	return &FileInfo{
		name: name,
		size: size,
		mod:  mod,
		mode: mode,
	}
}

// Name is base name of the file
func (fi *FileInfo) Name() string {
	return fi.name
}

// Size is length in bytes for regular files; system-dependent for others
func (fi *FileInfo) Size() int64 {
	return fi.size
}

// Mode is file mode bits
func (fi *FileInfo) Mode() fs.FileMode {
	return fi.mode
}

// ModTime is modification time
func (fi *FileInfo) ModTime() time.Time {
	return fi.mod
}

// IsDir abbreviation for Mode().IsDir()
func (fi *FileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys underlying data source (can return nil)
func (fi *FileInfo) Sys() interface{} {
	return nil
}
