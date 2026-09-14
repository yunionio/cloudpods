package rwfs

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

// in memory implementation of rwfs
// TODO: support fs.DirEntry, fs.ReadDirFile, fs.StatFS

type MemFS struct {
	base string
	root *MemDir
}

type MemChild interface{}
type MemDir struct {
	child map[string]MemChild
	mod   time.Time
}
type MemFile struct {
	b   []byte
	mod time.Time
}
type MemDirFP struct {
	f      *MemDir
	cur    int
	closed bool
	name   string
	flags  int
}
type MemFileFP struct {
	f      *MemFile
	cur    int
	closed bool
	name   string
	flags  int
}

func MemNew() *MemFS {
	return &MemFS{
		base: "",
		root: &MemDir{
			child: map[string]MemChild{},
		},
	}
}

func (o *MemFS) Create(name string) (WFile, error) {
	return o.OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

func (o *MemFS) Mkdir(name string, perm fs.FileMode) error {
	if name == "." {
		return fs.ErrExist
	}
	dir, base := path.Split(name)
	memDir, err := o.getDir(dir)
	if err != nil {
		return &fs.PathError{
			Op:   "mkdir",
			Path: dir,
			Err:  err,
		}
	}
	if _, ok := memDir.child[base]; ok {
		return &fs.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  fs.ErrExist,
		}
	}
	newDir := MemDir{mod: time.Now(), child: map[string]MemChild{}}
	memDir.child[base] = &newDir
	return nil
}

func (o *MemFS) OpenFile(name string, flags int, perm fs.FileMode) (RWFile, error) {
	dir, file := path.Split(name)
	memDir, err := o.getDir(dir)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}
	var child MemChild
	if file == "." || file == "" {
		child = memDir
	} else if _, ok := memDir.child[file]; ok {
		child = memDir.child[file]
	}
	if child != nil {
		switch v := child.(type) {
		case *MemFile:
			if flagSet(O_CREATE, flags) && flagSet(O_EXCL, flags) {
				return nil, &fs.PathError{
					Op:   "open",
					Path: name,
					Err:  fs.ErrExist,
				}
			}
			fp := MemFileFP{f: v, name: file, flags: flags}
			if flagSet(O_TRUNC, flags) {
				fp.f.b = []byte{}
				fp.f.mod = time.Now()
			}
			if flagSet(O_APPEND, flags) {
				fp.cur = len(fp.f.b)
			}
			return &fp, nil
		case *MemDir:
			if flagMode(flags) == O_WRONLY || flagMode(flags) == O_RDWR || flagSet(O_CREATE, flags) {
				return nil, &fs.PathError{
					Op:   "open",
					Path: name,
					Err:  fs.ErrExist,
				}
			}
			return &MemDirFP{f: v, name: file, flags: flags}, nil
		default:
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err:  fs.ErrInvalid,
			}
		}
	} else {
		// check for read and create flags
		if flagMode(flags) == O_RDONLY || !flagSet(O_CREATE, flags) {
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err:  fs.ErrNotExist,
			}
		}
		memFile := MemFile{mod: time.Now()}
		memDir.child[file] = &memFile
		memDir.mod = time.Now()
		return &MemFileFP{f: &memFile, name: file, flags: flags}, nil
	}
}

func (o *MemFS) Open(name string) (fs.File, error) {
	return o.OpenFile(name, O_RDONLY, 0)
}

func (o *MemFS) Remove(name string) error {
	if name == "." {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	dir, file := path.Split(name)
	memDir, err := o.getDir(dir)
	if err != nil {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  err,
		}
	}
	if child, ok := memDir.child[file]; ok {
		switch v := child.(type) {
		case *MemFile:
			// delete file
			delete(memDir.child, file)
			return nil
		case *MemDir:
			// check for contents of directory
			if len(v.child) > 0 {
				return &fs.PathError{
					Op:   "remove",
					Path: name,
					Err:  fmt.Errorf("directory not empty"),
				}
			}
			delete(memDir.child, file)
			return nil
		default:
			return &fs.PathError{
				Op:   "remove",
				Path: name,
				Err:  fs.ErrInvalid,
			}
		}
	} else {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}
}

// Rename moves a file or directory to a new name
func (o *MemFS) Rename(oldName, newName string) error {
	dirOld, fileOld := path.Split(oldName)
	memDirOld, err := o.getDir(dirOld)
	if err != nil {
		return &fs.PathError{
			Op:   "rename",
			Path: oldName,
			Err:  err,
		}
	}
	dirNew, fileNew := path.Split(newName)
	memDirNew, err := o.getDir(dirNew)
	if err != nil {
		return &fs.PathError{
			Op:   "rename",
			Path: newName,
			Err:  err,
		}
	}
	childOld, okOld := memDirOld.child[fileOld]
	if !okOld {
		return &fs.PathError{
			Op:   "rename",
			Path: oldName,
			Err:  fs.ErrNotExist,
		}
	}
	childNew, okNew := memDirNew.child[fileNew]

	switch vOld := childOld.(type) {
	case *MemFile:
		// source is a file
		if !okNew {
			// new name doesn't already exist
			memDirNew.child[fileNew] = vOld
		} else {
			switch childNew.(type) {
			case *MemFile:
				// replacing a file
				memDirNew.child[fileNew] = vOld
			case *MemDir:
				// reject for now, raise an issue if this should be allowed
				return &fs.PathError{
					Op:   "rename",
					Path: newName,
					Err:  fs.ErrExist,
				}
			default:
				return &fs.PathError{
					Op:   "rename",
					Path: newName,
					Err:  fs.ErrInvalid,
				}
			}
		}
	case *MemDir:
		// source is a directory
		if !okNew {
			// new name doesn't already exist, move folder
			memDirNew.child[fileNew] = vOld
		} else {
			switch childNew.(type) {
			case *MemFile:
				// copy directory to a file, invalid
				return &fs.PathError{
					Op:   "rename",
					Path: newName,
					Err:  fs.ErrExist,
				}
			case *MemDir:
				// copy a directory to a directory
				// for now consider this invalid, raise an issue with a scenario to permit
				return &fs.PathError{
					Op:   "rename",
					Path: newName,
					Err:  fs.ErrExist,
				}
			default:
				return &fs.PathError{
					Op:   "rename",
					Path: newName,
					Err:  fs.ErrInvalid,
				}
			}
		}
	default:
		return &fs.PathError{
			Op:   "remove",
			Path: oldName,
			Err:  fs.ErrInvalid,
		}
	}
	// remove old directory entry
	delete(memDirOld.child, fileOld)
	return nil
}

func (o *MemFS) Sub(name string) (*MemFS, error) {
	if name == "." {
		return o, nil
	}
	full, err := o.join("sub", name)
	if err != nil {
		return nil, err
	}
	subRoot, err := o.getDir(name)
	if err != nil {
		return nil, err
	}
	return &MemFS{
		base: full,
		root: subRoot,
	}, nil
}

func (o *MemFS) join(op, name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{
			Op:   op,
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	return path.Join(o.base, name), nil
}

func (o *MemFS) getDir(dir string) (*MemDir, error) {
	cur := o.root
	if dir == "" || dir == "." {
		return cur, nil
	}
	dir = strings.TrimSuffix(dir, "/")
	for _, el := range strings.Split(dir, "/") {
		if el == "." || el == "" {
			continue
		}
		next, ok := cur.child[el]
		if !ok {
			return nil, fs.ErrNotExist
		}
		nextDir, ok := next.(*MemDir)
		if !ok {
			return nil, fs.ErrInvalid
		}
		cur = nextDir
	}
	return cur, nil
}

func (mfp *MemFileFP) Close() error {
	mfp.closed = true
	return nil
}

func (mfp *MemFileFP) Read(b []byte) (int, error) {
	lc := copy(b, mfp.f.b[mfp.cur:])
	mfp.cur += lc
	if len(mfp.f.b) <= mfp.cur {
		return lc, io.EOF
	}
	return lc, nil
}

func (mfp *MemFileFP) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		mfp.cur = whence
	case io.SeekEnd:
		mfp.cur = int(int64(len(mfp.f.b)) + offset)
	case io.SeekCurrent:
		mfp.cur += int(offset)
	default:
		return -1, fmt.Errorf("unknown whence value: %d", whence)
	}
	return int64(mfp.cur), nil
}

func (mfp *MemFileFP) Stat() (fs.FileInfo, error) {
	fi := NewFI(mfp.name, int64(len(mfp.f.b)), time.Time{}, 0)
	return fi, nil
}

func (mfp *MemFileFP) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}
	// use copy to overwrite existing contents
	if mfp.cur < len(mfp.f.b) {
		l := copy(mfp.f.b[mfp.cur:], b)
		if l < len(b) {
			mfp.f.b = append(mfp.f.b, b[l:]...)
		}
	} else {
		mfp.f.b = append(mfp.f.b, b...)
	}
	mfp.cur += len(b)
	mfp.f.mod = time.Now()
	return len(b), nil
}

func (mdp *MemDirFP) Close() error {
	mdp.closed = true
	return nil
}

func (mdp *MemDirFP) Read(b []byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: mdp.name,
		Err:  fs.ErrInvalid,
	}
}

// TODO: implement func (mdp *MemDirFP) Seek

func (mdp *MemDirFP) ReadDir(n int) ([]fs.DirEntry, error) {
	names := mdp.filenames(mdp.cur, n)
	mdp.cur += len(names)
	des := make([]fs.DirEntry, len(names))
	for i, name := range names {
		var size int64
		var mod time.Time
		var mode fs.FileMode
		switch v := mdp.f.child[name].(type) {
		case *MemFile:
			size = int64(len(v.b))
			mod = v.mod
			mode = 0
		case *MemDir:
			size = int64(len(v.child))
			mod = v.mod
			mode = fs.ModeDir
		default:
			return des[:0], &fs.PathError{
				Op:   "readdir",
				Path: path.Join(mdp.name, name),
				Err:  fs.ErrInvalid,
			}
		}
		fi := NewFI(name, size, mod, mode)
		des[i] = NewDE(name, fi.mode, fi)
	}

	if (n > 0 && len(names) < n) || (n == 0 && len(names) == 0) {
		return des, io.EOF
	} else {
		return des, nil
	}
}

func (mdp *MemDirFP) Stat() (fs.FileInfo, error) {
	fi := NewFI(mdp.name, 4096, mdp.f.mod, fs.ModeDir)
	return fi, nil
}

func (mdp *MemDirFP) Write(b []byte) (n int, err error) {
	return 0, &fs.PathError{
		Op:   "write",
		Path: mdp.name,
		Err:  fs.ErrInvalid,
	}
}

func (mdp *MemDirFP) filenames(start, limit int) []string {
	// get a sorted list of keys from the map
	names := make([]string, len(mdp.f.child))
	i := 0
	for k := range mdp.f.child {
		names[i] = k
		i++
	}
	sort.Strings(names)
	// return the appropriate slice from that map
	if limit <= 0 {
		return names[start:]
	} else {
		end := start + limit
		if end > len(names) {
			end = len(names)
		}
		return names[start:end]
	}
}
