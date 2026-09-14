// Package rwfs implements a read-write filesystem, extending fs.FS
package rwfs

import (
	"errors"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

//lint:file-ignore ST1003 names are uppercase to remain compatible with os names
const (
	// exactly one of these must be used
	O_RDONLY = os.O_RDONLY // read-only
	O_WRONLY = os.O_WRONLY // write-only
	O_RDWR   = os.O_RDWR   // read-write
	// remaining values may be or'ed
	O_APPEND = os.O_APPEND // append when writing
	O_CREATE = os.O_CREATE // create if missing
	O_EXCL   = os.O_EXCL   // file must not exist, used with O_CREATE
	O_SYNC   = os.O_SYNC   // synchronous I/O
	O_TRUNC  = os.O_TRUNC  // truncate on open
)

type RWFS interface {
	fs.FS
	WriteFS
}
type RWFile interface {
	fs.File
	WFile
}
type RWPerms interface {
	Chmod(filename string, mode fs.FileMode) error
	Chown(filename string, uid, gid int) error
}

// WriteFS is an interface for a writable filesystem
type WriteFS interface {
	// Create creates a new file
	Create(string) (WFile, error)
	// Mkdir creates a directory
	Mkdir(string, fs.FileMode) error
	// OpenFile generalized file open with options for a flag and permissions
	OpenFile(string, int, fs.FileMode) (RWFile, error)
	// Remove removes the named file or (empty) directory.
	Remove(string) error
	// Rename moves a file or directory to a new name
	Rename(oldName, newName string) error
}

// WFile is the interface for a writable file
type WFile interface {
	// Close closes the open file
	Close() error
	// Stat returns the FileInfo of the file
	Stat() (fi fs.FileInfo, err error)
	// Write writes len(b) bytes to the file.
	// It returns the number of bytes written, and any error if n != len(b).
	Write(b []byte) (n int, err error)
}

// Copy will copy a file to a new name, and even a different rwfs
func Copy(srcFS fs.FS, srcName string, destFS RWFS, destName string) error {
	rfh, err := srcFS.Open(srcName)
	if err != nil {
		return err
	}
	defer rfh.Close()
	wfh, err := destFS.Create(destName)
	if err != nil {
		return err
	}
	defer wfh.Close()
	_, err = io.Copy(wfh, rfh)
	return err
}

// CopyRecursive will recursively copy a directory tree to the destination
func CopyRecursive(srcFS fs.FS, srcName string, destFS RWFS, destName string) error {
	sfh, err := srcFS.Open(srcName)
	if err != nil {
		return err
	}
	sfi, err := sfh.Stat()
	sfh.Close()
	if err != nil {
		return err
	}
	if !sfi.IsDir() {
		return Copy(srcFS, srcName, destFS, destName)
	}

	sdReadDir, err := fs.ReadDir(srcFS, srcName)
	if err != nil {
		return err
	}
	err = destFS.Mkdir(destName, 0777)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	for _, de := range sdReadDir {
		srcNew := path.Join(srcName, de.Name())
		destNew := path.Join(destName, de.Name())
		err = CopyRecursive(srcFS, srcNew, destFS, destNew)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateTemp returns a temp file
func CreateTemp(rwfs RWFS, dir, pattern string) (RWFile, error) {
	ct, ok := rwfs.(interface {
		CreateTemp(dir, pattern string) (RWFile, error)
	})
	if ok {
		return ct.CreateTemp(dir, pattern)
	}
	if dir == "" {
		dir = os.TempDir()
	}
	prefix, suffix := pattern, ""
	i := strings.LastIndex(pattern, "*")
	if i >= 0 {
		prefix, suffix = pattern[:i], pattern[i+1:]
	}
	prefix = filepath.Join(dir, prefix)
	try := 0
	for {
		rnd := strconv.FormatUint(rand.Uint64(), 10)
		name := prefix + rnd + suffix
		f, err := rwfs.OpenFile(name, O_RDWR|O_CREATE|O_EXCL, 0600)
		if err != nil && errors.Is(err, fs.ErrExist) {
			try++
			if try < 10000 {
				continue
			}
			return nil, &fs.PathError{Op: "createtemp", Path: prefix + "*" + suffix, Err: fs.ErrExist}
		}
		return f, err
	}
}

// MkdirAll creates a directory, including all parent directories
func MkdirAll(rwfs RWFS, name string, perm fs.FileMode) error {
	parts := strings.Split(name, "/")
	prefix := ""
	if strings.HasPrefix(name, "/") {
		prefix = "/"
	}
	for i := range parts {
		// assemble directory up to this point
		cur := prefix + path.Join(parts[:i+1]...)
		if cur == "" || cur == "." || cur == "/" {
			continue
		}
		fi, err := Stat(rwfs, cur)
		if errors.Is(err, fs.ErrNotExist) {
			// missing, create
			err := rwfs.Mkdir(cur, perm)
			if err != nil {
				return &fs.PathError{
					Op:   "mkdir",
					Path: cur,
					Err:  err,
				}
			}
		} else if err != nil {
			// unknown errors
			return &fs.PathError{
				Op:   "mkdir",
				Path: cur,
				Err:  err,
			}
		} else if !fi.IsDir() {
			// can't mkdir on existing file
			return &fs.PathError{
				Op:   "mkdir",
				Path: cur,
				Err:  fs.ErrExist,
			}
		}
		// exists and is a directory, next
	}
	return nil
}

// TODO: add Rename func

// Stat returns the FileInfo for a specified file
func Stat(rfs fs.FS, name string) (fs.FileInfo, error) {
	fh, err := rfs.Open(name)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return nil, err
	}
	return fi, nil
}

// ReadFile returns the file contents
func ReadFile(rfs fs.FS, name string) ([]byte, error) {
	return fs.ReadFile(rfs, name)
}

// WriteFile replaces or creates a file with the specified contents
func WriteFile(wfs WriteFS, name string, data []byte, perm fs.FileMode) error {
	// replace os flags?
	f, err := wfs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		return err1
	}
	return err
}

func flagMode(flags int) int {
	return flags & (O_RDONLY | O_WRONLY | O_RDWR)
}

func flagSet(flag, flags int) bool {
	return (flags & flag) != 0
}
