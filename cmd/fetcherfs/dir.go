package main

import (
	"context"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/pkg/errors"
)

const (
	CONTENT_FILE_NAME = "content"
	META_FILE_NAME    = "meta"
)

// Dir implements both Node and Handle for the root directory.
type Dir struct{}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0755
	return nil
}

func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == CONTENT_FILE_NAME {
		return Content{}, nil
	} else if name == META_FILE_NAME {
		return Meta{}, nil
	}
	return nil, syscall.ENOENT
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return dirDirs, nil
}

var (
	dirDirs = []fuse.Dirent{
		{Inode: 2, Name: CONTENT_FILE_NAME, Type: fuse.DT_File},
		{Inode: 3, Name: META_FILE_NAME, Type: fuse.DT_File},
	}
)

// Content implements both Node and Handle for the content file.
type Content struct{}

func NewContent() *Content {
	return &Content{}
}

func (Content) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 2
	a.Mode = 0444
	a.Size = uint64(fetcherFs.GetSize())
	a.BlockSize = uint32(opt.Blocksize) * 1024 * 1024
	return nil
}

func (Content) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// fmt.Printf("req.Size %d, req.Offset %d, resp.Data %d", req.Size, req.Offset, len(resp.Data))
	if req.Offset >= 0 && req.Offset < fetcherFs.GetSize() {
		if req.Offset+int64(req.Size) > fetcherFs.GetSize() {
			req.Size = int(fetcherFs.GetSize() - req.Offset)
		}
		if data, err := fetcherFs.doRead(req.Size, req.Offset); err != nil {
			return err
		} else {
			resp.Data = data
			return nil
		}
	} else if req.Offset == fetcherFs.GetSize() {
		return nil
	} else {
		return errors.Errorf("bad offset %d", req.Offset)
	}
}

// Meta implements both Node and Handle for the meta file.
type Meta struct{}

func NewMeta() *Meta {
	return &Meta{}
}

func (Meta) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 3
	a.Mode = 0444
	a.Size = uint64(len(fetcherFs.GetMeta()))
	return nil
}

func (Meta) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(fetcherFs.GetMeta()), nil
}
