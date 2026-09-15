package rwfs

import (
	"io/fs"
	"os"
	"path"
)

// RWFS implemented for the os filesystem
// TODO: support fs.DirEntry, fs.ReadDirFile, fs.StatFS

type OSFS struct {
	dir string
}

// OSFile is a wrapper around os.* to implement RWFS
type OSFile struct {
	*os.File
}

func OSNew(base string) *OSFS {
	if base == "" || base == "." {
		return &OSFS{dir: base}
	}
	base = path.Clean(base)
	return &OSFS{
		dir: base,
	}
}

func (o *OSFS) Chmod(name string, mode fs.FileMode) error {
	file, err := o.join("chmod", name)
	if err != nil {
		return err
	}
	return os.Chmod(file, mode)
}

func (o *OSFS) Chown(name string, uid, gid int) error {
	file, err := o.join("chown", name)
	if err != nil {
		return err
	}
	return os.Chown(file, uid, gid)
}

func (o *OSFS) Create(name string) (WFile, error) {
	file, err := o.join("create", name)
	if err != nil {
		return nil, err
	}
	fh, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	return &OSFile{
		File: fh,
	}, nil
}

func (o *OSFS) CreateTemp(dir, pattern string) (RWFile, error) {
	dir, err := o.join("createtemp", dir)
	if err != nil {
		return nil, err
	}
	fh, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &OSFile{
		File: fh,
	}, nil
}

func (o *OSFS) Mkdir(name string, perm fs.FileMode) error {
	if name == "." {
		return fs.ErrExist
	}
	dir, err := o.join("mkdir", name)
	if err != nil {
		return err
	}
	return os.Mkdir(dir, perm)
}

func (o *OSFS) OpenFile(name string, flag int, perm fs.FileMode) (RWFile, error) {
	file, err := o.join("open", name)
	if err != nil {
		return nil, err
	}
	fh, err := os.OpenFile(file, flag, perm)
	if err != nil {
		return nil, err
	}
	return &OSFile{
		File: fh,
	}, nil
}

func (o *OSFS) Open(name string) (fs.File, error) {
	file, err := o.join("open", name)
	if err != nil {
		return nil, err
	}
	fh, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return &OSFile{
		File: fh,
	}, nil
}

func (o *OSFS) Remove(name string) error {
	if name == "." {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	full, err := o.join("remove", name)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

// Rename moves a file or directory to a new name
func (o *OSFS) Rename(oldName, newName string) error {
	oldFile, err := o.join("rename", oldName)
	if err != nil {
		return err
	}
	newFile, err := o.join("rename", newName)
	if err != nil {
		return err
	}
	return os.Rename(oldFile, newFile)
}

func (o *OSFS) Sub(name string) (*OSFS, error) {
	if name == "." {
		return o, nil
	}
	full, err := o.join("sub", name)
	if err != nil {
		return nil, err
	}
	return &OSFS{
		dir: full,
	}, nil
}

func (o *OSFS) join(op, name string) (string, error) {
	if name == "" || name == "." {
		if o.dir != "" {
			return o.dir, nil
		}
		return ".", nil
	}
	// relative paths allowed when o.dir is not set
	if o.dir == "" {
		return path.Clean(name), nil
	}
	// clean path to prevent traversing outside of o.dir
	if name[:1] == "/" {
		name = path.Clean(name)
	} else {
		name = path.Clean("/" + name)
		if len(name) > 1 {
			name = name[1:]
		}
	}
	if !fs.ValidPath(name) {
		return "", &fs.PathError{
			Op:   op,
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	return path.Join(o.dir, name), nil
}
