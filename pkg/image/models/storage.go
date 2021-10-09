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

package models

import (
	"fmt"
	"io"
	"os"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/image/drivers/s3"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var local Storage = &LocalStorage{}
var s3Instance Storage = &S3Storage{}
var storage Storage

func GetStorage() Storage {
	return storage
}

func GetImage(location string) (int64, io.ReadCloser, error) {
	switch {
	case strings.HasPrefix(location, image.S3Prefix):
		return s3Instance.GetImage(location[len(image.S3Prefix):])
	case strings.HasPrefix(location, image.LocalFilePrefix):
		return local.GetImage(location[len(image.LocalFilePrefix):])
	default:
		return local.GetImage(location)
	}
}

func RemoveImage(location string) error {
	switch {
	case strings.HasPrefix(location, image.S3Prefix):
		return s3Instance.RemoveImage(location[len(image.S3Prefix):])
	case strings.HasPrefix(location, image.LocalFilePrefix):
		return local.RemoveImage(location[len(image.LocalFilePrefix):])
	default:
		return local.RemoveImage(location)
	}
}

func IsCheckStatusEnabled(img *SImage) bool {
	switch {
	case strings.HasPrefix(img.Location, image.S3Prefix):
		return s3Instance.IsCheckStatusEnabled()
	case strings.HasPrefix(img.Location, image.LocalFilePrefix):
		return local.IsCheckStatusEnabled()
	default:
		return local.IsCheckStatusEnabled()
	}
}

func Init(storageBackend string) {
	switch storageBackend {
	case image.IMAGE_STORAGE_DRIVER_LOCAL:
		storage = &LocalStorage{}
	case image.IMAGE_STORAGE_DRIVER_S3:
		storage = &S3Storage{}
	default:
		storage = &LocalStorage{}
	}
}

type Storage interface {
	Type() string
	SaveImage(string) (string, error)
	CleanTempfile(string) error
	GetImage(string) (int64, io.ReadCloser, error)
	RemoveImage(string) error

	IsCheckStatusEnabled() bool
}

type LocalStorage struct{}

func (s *LocalStorage) Type() string {
	return image.IMAGE_STORAGE_DRIVER_LOCAL
}

func (s *LocalStorage) SaveImage(imagePath string) (string, error) {
	return fmt.Sprintf("%s%s", LocalFilePrefix, imagePath), nil
}

func (s *LocalStorage) CleanTempfile(filePath string) error {
	return nil
}

func (s *LocalStorage) GetImage(imagePath string) (int64, io.ReadCloser, error) {
	fstat, err := os.Stat(imagePath)
	if err != nil {
		return -1, nil, errors.Wrapf(err, "stat file %s", imagePath)
	}
	f, err := os.Open(imagePath)
	if err != nil {
		return -1, nil, errors.Wrapf(err, "open file %s", imagePath)
	}
	return fstat.Size(), f, nil
}

func (s *LocalStorage) IsCheckStatusEnabled() bool {
	return true
}

func (s *LocalStorage) RemoveImage(imagePath string) error {
	return os.Remove(imagePath)
}

type S3Storage struct{}

func imagePathToName(imagePath string) string {
	segs := strings.Split(imagePath, "/")
	return segs[len(segs)-1]
}

func (s *S3Storage) Type() string {
	return image.IMAGE_STORAGE_DRIVER_S3
}

func (s *S3Storage) SaveImage(imagePath string) (string, error) {
	return s3.Put(imagePath, imagePathToName(imagePath))
}

func (s *S3Storage) CleanTempfile(filePath string) error {
	out, err := procutils.NewCommand("rm", "-f", filePath).Output()
	if err != nil {
		return errors.Wrapf(err, "rm %s failed %s", filePath, out)
	}
	return nil
}

func (s *S3Storage) GetImage(imagePath string) (int64, io.ReadCloser, error) {
	obj, err := s3.Get(imagePathToName(imagePath))
	if err != nil {
		return -1, nil, errors.Wrap(err, "s3 get image")
	}
	objInfo, err := obj.Stat()
	if err != nil {
		return -1, nil, errors.Wrap(err, "s3 obj stat")
	}
	return objInfo.Size, obj, nil
}

func (s *S3Storage) IsCheckStatusEnabled() bool {
	return options.Options.S3CheckImageStatus
}

func (s *S3Storage) RemoveImage(fileName string) error {
	return s3.Remove(fileName)
}
