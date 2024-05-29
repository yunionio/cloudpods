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
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/image/drivers/s3"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var local IImageStorage = &LocalStorage{}
var s3Instance IImageStorage = &S3Storage{}
var storage IImageStorage

func GetStorage() IImageStorage {
	return storage
}

func GetImage(ctx context.Context, location string) (int64, io.ReadCloser, error) {
	switch {
	case strings.HasPrefix(location, image.S3Prefix):
		return s3Instance.GetImage(ctx, location[len(image.S3Prefix):])
	case strings.HasPrefix(location, image.LocalFilePrefix):
		return local.GetImage(ctx, location[len(image.LocalFilePrefix):])
	default:
		return local.GetImage(ctx, location)
	}
}

func RemoveImage(ctx context.Context, location string) error {
	switch {
	case strings.HasPrefix(location, image.S3Prefix):
		return s3Instance.RemoveImage(ctx, location[len(image.S3Prefix):])
	case strings.HasPrefix(location, image.LocalFilePrefix):
		return local.RemoveImage(ctx, location[len(image.LocalFilePrefix):])
	default:
		return local.RemoveImage(ctx, location)
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

type IImageStorage interface {
	Type() string
	SaveImage(context.Context, string, func(int64)) (string, error)
	CleanTempfile(string) error
	GetImage(context.Context, string) (int64, io.ReadCloser, error)
	RemoveImage(context.Context, string) error

	IsCheckStatusEnabled() bool
	ConvertImage(ctx context.Context, image *SImage, targetFormat string, progresser func(saved int64)) (*SConverImageInfo, error)
}

type LocalStorage struct{}

func (s *LocalStorage) Type() string {
	return image.IMAGE_STORAGE_DRIVER_LOCAL
}

func (s *LocalStorage) SaveImage(ctx context.Context, imagePath string, progresser func(saved int64)) (string, error) {
	return fmt.Sprintf("%s%s", LocalFilePrefix, imagePath), nil
}

func (s *LocalStorage) CleanTempfile(filePath string) error {
	return nil
}

func (s *LocalStorage) GetImage(ctx context.Context, imagePath string) (int64, io.ReadCloser, error) {
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

func (s *LocalStorage) ConvertImage(ctx context.Context, image *SImage, targetFormat string, progresser func(saved int64)) (*SConverImageInfo, error) {
	location := image.GetPath(targetFormat)
	img, err := image.getQemuImage()
	if err != nil {
		return nil, errors.Wrap(err, "unable to image.getQemuImage")
	}
	nimg, err := img.Clone(location, qemuimgfmt.String2ImageFormat(targetFormat), true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to img.Clone")
	}
	return &SConverImageInfo{
		Location:  fmt.Sprintf("%s%s", LocalFilePrefix, location),
		SizeBytes: nimg.ActualSizeBytes,
	}, nil
}

func (s *LocalStorage) IsCheckStatusEnabled() bool {
	return true
}

func (s *LocalStorage) RemoveImage(ctx context.Context, imagePath string) error {
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

func (s *S3Storage) SaveImage(ctx context.Context, imagePath string, progresser func(saved int64)) (string, error) {
	if !fileutils2.IsFile(imagePath) {
		return "", fmt.Errorf("%s not valid file", imagePath)
	}
	return s3.Put(ctx, imagePath, imagePathToName(imagePath), progresser)
}

func (s *S3Storage) CleanTempfile(filePath string) error {
	out, err := procutils.NewCommand("rm", "-f", filePath).Output()
	if err != nil {
		return errors.Wrapf(err, "rm %s failed %s", filePath, out)
	}
	return nil
}

func (s *S3Storage) getTempDir() (string, error) {
	var dir string
	if options.Options.FilesystemStoreDatadir != "" {
		dir = options.Options.FilesystemStoreDatadir + "/image-tmp"
	} else {
		dir = "/tmp/image-tmp"
	}
	if !fileutils2.Exists(dir) {
		err := procutils.NewCommand("mkdir", "-p", dir).Run()
		if err != nil {
			return "", errors.Wrapf(err, "unable to create dir %s", dir)
		}
	}
	return dir, nil
}

type SConverImageInfo struct {
	Location  string
	SizeBytes int64
}

func (s *S3Storage) ConvertImage(ctx context.Context, image *SImage, targetFormat string, progresser func(saved int64)) (*SConverImageInfo, error) {
	tempDir, err := s.getTempDir()
	if err != nil {
		return nil, err
	}
	location := fmt.Sprintf("%s/%s.%s", tempDir, image.GetId(), targetFormat)
	img, err := image.getQemuImage()
	if err != nil {
		return nil, errors.Wrap(err, "unable to image.getQemuImage")
	}
	nimg, err := img.Clone(location, qemuimgfmt.String2ImageFormat(targetFormat), true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to img.Clone")
	}
	defer s.CleanTempfile(location)
	s3Location, err := s.SaveImage(ctx, location, progresser)
	if err != nil {
		return nil, errors.Wrap(err, "unable to SaveImage")
	}
	return &SConverImageInfo{
		Location:  s3Location,
		SizeBytes: nimg.ActualSizeBytes,
	}, nil
}

func (s *S3Storage) GetImage(ctx context.Context, imagePath string) (int64, io.ReadCloser, error) {
	return s3.Get(ctx, imagePathToName(imagePath))
}

func (s *S3Storage) IsCheckStatusEnabled() bool {
	return options.Options.S3CheckImageStatus
}

func (s *S3Storage) RemoveImage(ctx context.Context, fileName string) error {
	return s3.Remove(ctx, fileName)
}
