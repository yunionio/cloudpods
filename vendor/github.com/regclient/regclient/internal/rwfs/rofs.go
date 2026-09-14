package rwfs

import "io/fs"

// ROFS wraps a fs.FS with permission denied on any write attempt
type ROFS struct {
	fs.FS // read requests pass through to underlying FS and File
}
type ROFile struct {
	fs.File
}

type roConfig struct {
	roFS fs.FS
}

// ROOpts specifies options for RONew
type ROOpts func(*roConfig)

// RONew creates a new read-only filesystem
func RONew(opts ...ROOpts) *ROFS {
	rc := roConfig{}
	for _, opt := range opts {
		opt(&rc)
	}
	if rc.roFS == nil {
		return nil
	}
	return &ROFS{
		FS: rc.roFS,
	}
}

// WithROFS provides the fs.FS used by ROFS
func WithROFS(roFS fs.FS) ROOpts {
	return func(fc *roConfig) {
		fc.roFS = roFS
	}
}

func (rofs *ROFS) Create(name string) (*ROFile, error) {
	return nil, &fs.PathError{
		Op:   "create",
		Path: name,
		Err:  fs.ErrPermission,
	}
}

func (rofs *ROFS) Mkdir(name string, perm fs.FileMode) error {
	return &fs.PathError{
		Op:   "mkdir",
		Path: name,
		Err:  fs.ErrPermission,
	}
}

func (rofs *ROFS) Open(name string) (*ROFile, error) {
	fp, err := rofs.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &ROFile{File: fp}, nil
}

func (rofs *ROFS) OpenFile(name string, flag int, perm fs.FileMode) (*ROFile, error) {
	// TODO: support open for read
	return nil, &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  fs.ErrPermission,
	}
}

func (rof *ROFile) Write(b []byte) (n int, err error) {
	return 0, &fs.PathError{
		Op:  "write",
		Err: fs.ErrPermission,
	}
}
