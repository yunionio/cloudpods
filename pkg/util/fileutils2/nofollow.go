// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fileutils2

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"yunion.io/x/pkg/errors"
)

// IsPathInside reports whether target is the same as base or a descendant of it.
func IsPathInside(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

// CleanGuestDeployPath requires an absolute guest path and rejects parent-dir
// traversal after cleaning.
func CleanGuestDeployPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.Errorf("path is empty")
	}
	if strings.ContainsRune(p, 0) {
		return "", errors.Errorf("path contains NUL")
	}
	if !path.IsAbs(p) && !filepath.IsAbs(p) {
		return "", errors.Errorf("path %q must be absolute", p)
	}
	clean := path.Clean("/" + strings.TrimPrefix(filepath.ToSlash(p), "/"))
	if !path.IsAbs(clean) {
		return "", errors.Errorf("path %q must be absolute", p)
	}
	if clean == "/" {
		return "", errors.Errorf("path %q is not allowed", p)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.Errorf("path %q is not allowed", p)
	}
	return clean, nil
}

func IsSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

func OpenFileNoFollow(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag|syscall.O_NOFOLLOW, perm)
}

func FilePutContentsNoFollow(filename string, content string, modAppend bool) error {
	mode := os.O_WRONLY | os.O_CREATE
	if modAppend {
		mode |= os.O_APPEND
	} else {
		mode |= os.O_TRUNC
	}
	fd, err := OpenFileNoFollow(filename, mode, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(content)
	return err
}

func FileGetContentsNoFollow(filename string) ([]byte, error) {
	fd, err := OpenFileNoFollow(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return io.ReadAll(fd)
}
