// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cp

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type ICopy interface {
	CopyToContainer(s *mcclient.ClientSession, srcPath string, dest ContainerFileOpt) error
	CopyFromContainer(s *mcclient.ClientSession, src ContainerFileOpt, destPath string) error
}

type sCopy struct {
	noPreserve        bool
	ExecParentCmdName string
}

func NewCopy() ICopy {
	return &sCopy{}
}

type ContainerFileOpt struct {
	ContainerId string
	File        string
}

var ErrFileCannotBeEmpty = errors.Error("filepath can not be empty")

func (o *sCopy) CopyFromContainer(s *mcclient.ClientSession, src ContainerFileOpt, destPath string) error {
	if len(src.File) == 0 || len(destPath) == 0 {
		return ErrFileCannotBeEmpty
	}

	reader, outStream := io.Pipe()
	go func() {
		defer outStream.Close()
		if err := compute.Containers.CopyTarFrom(s, src.ContainerId, []string{src.File}, outStream); err != nil {
			log.Errorf("copy src by tar from container: %v", err)
		}
	}()

	prefix := getPrefix(src.File)
	prefix = path.Clean(prefix)
	// remove extraneous path shortcuts - these could occur if a path contained extra "../
	// and attempted to navigate beyond "/" in a remote filesystem
	prefix = stripPathShortcuts(prefix)
	return o.untarAll(src, reader, destPath, prefix)
}

func getPrefix(file string) string {
	// tar strips the leading '/' if it's there, so we will too
	return strings.TrimLeft(file, "/")
}

// stripPathShortcuts removes any leading or trailing "../" from a given path
func stripPathShortcuts(p string) string {
	newPath := path.Clean(p)
	trimmed := strings.TrimPrefix(newPath, "../")

	for trimmed != newPath {
		newPath = trimmed
		trimmed = strings.TrimPrefix(newPath, "../")
	}

	// trim leftover {".", ".."}
	if newPath == "." || newPath == ".." {
		newPath = ""
	}

	if len(newPath) > 0 && string(newPath[0]) == "/" {
		return newPath[1:]
	}

	return newPath
}

func (o *sCopy) untarAll(src ContainerFileOpt, reader io.Reader, destDir, prefix string) error {
	symlinkWarningPrinted := false
	// TODO: use compression here?
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		// All the files will start with the prefix, which is the directory where
		// they were located on the pod, we need to strip down that prefix, but
		// if the prefix is missing it means the tar was tempered with.
		// For the case where prefix is empty we need to ensure that the path
		// is not absolute, which also indicates the tar file was tempered with.
		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("tar contents corrupted")
		}

		// basic file information
		mode := header.FileInfo().Mode()
		destFileName := filepath.Join(destDir, header.Name[len(prefix):])

		if !isDestRelative(destDir, destFileName) {
			fmt.Fprintf(os.Stderr, "warning: file %q is outside target destination, skipping\n", destFileName)
			continue
		}

		baseName := filepath.Dir(destFileName)
		if err := os.MkdirAll(baseName, 0755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(destFileName, 0755); err != nil {
				return err
			}
			continue
		}

		if mode&os.ModeSymlink != 0 {
			if !symlinkWarningPrinted && len(o.ExecParentCmdName) > 0 {
				fmt.Fprintf(os.Stderr, "warning: skipping symlink: %q -> %q\n", destFileName, header.Linkname)
				symlinkWarningPrinted = true
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: skipping symlink: %q -> %q\n", destFileName, header.Linkname)
			continue
		}
		outFile, err := os.Create(destFileName)
		if err != nil {
			return err
		}
		defer outFile.Close()
		if _, err := io.Copy(outFile, tarReader); err != nil {
			return err
		}
		if err := outFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

// isDestRelative returns true if dest is pointing outside the base directory,
// false otherwise.
func isDestRelative(base, dest string) bool {
	relative, err := filepath.Rel(base, dest)
	if err != nil {
		return false
	}
	return relative == "." || relative == stripPathShortcuts(relative)
}

func (o *sCopy) CopyToContainer(s *mcclient.ClientSession, srcFile string, dest ContainerFileOpt) error {
	if len(srcFile) == 0 || len(dest.File) == 0 {
		return ErrFileCannotBeEmpty
	}
	if _, err := os.Stat(srcFile); err != nil {
		return errors.Wrapf(err, "check source file: %s", srcFile)
	}
	reader, writer := io.Pipe()
	// strip trailing slash (if any)
	if dest.File != "/" && strings.HasSuffix(string(dest.File[len(dest.File)-1]), "/") {
		dest.File = dest.File[:len(dest.File)-1]
	}
	if err := compute.Containers.CheckDestinationIsDir(s, dest.ContainerId, dest.File); err == nil {
		// If no error, dest.File was found to be a directory.
		// Copy specified src info it
		dest.File = dest.File + "/" + path.Base(srcFile)
	}

	go func() {
		defer writer.Close()
		if err := makeTar(srcFile, dest.File, writer); err != nil {
			log.Errorf("makeTar error: %v", err)
		}
	}()

	destDir := path.Dir(dest.File)
	return compute.Containers.CopyTarTo(s, dest.ContainerId, destDir, reader, o.noPreserve)
}

func makeTar(srcPath, destPath string, writer io.Writer) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	srcPath := path.Join(srcBase, srcFile)
	matchedPaths, err := filepath.Glob(srcPath)
	if err != nil {
		return err
	}
	for _, fpath := range matchedPaths {
		stat, err := os.Lstat(fpath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := ioutil.ReadDir(fpath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := tar.FileInfoHeader(stat, fpath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := tar.FileInfoHeader(stat, fpath)
			target, err := os.Readlink(fpath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := tar.FileInfoHeader(stat, fpath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}
