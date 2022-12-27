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

package azure

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	DefaultStorageAccount string = "image"
	DefaultContainer      string = "image-cache"

	DefaultReadBlockSize int64 = 4 * 1024 * 1024
)

type SStoragecache struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Name, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetGlobalId())
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := self.region.GetImages(cloudprovider.ImageTypeCustomized)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i++ {
		images[i].storageCache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	img, err := self.region.GetImageById(extId)
	if err != nil {
		return nil, err
	}
	img.storageCache = self
	return &img, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	err := os.MkdirAll(image.TmpPath, os.ModePerm)
	if err != nil {
		log.Warningf("failed to create tmp path %s error: %v", image.TmpPath, err)
	}
	return self.uploadImage(ctx, image, image.TmpPath, callback)
}

func (self *SStoragecache) checkStorageAccount() (*SStorageAccount, error) {
	storageaccounts, err := self.region.ListStorageAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "ListStorageAccounts")
	}
	if len(storageaccounts) == 0 {
		storageaccount, err := self.region.CreateStorageAccount(self.region.Name)
		if err != nil {
			return nil, errors.Wrap(err, "CreateStorageAccount")
		}
		return storageaccount, nil
	}
	for i := 0; i < len(storageaccounts); i++ {
		if id, ok := storageaccounts[i].Tags["id"]; ok && id == self.region.Name {
			return &storageaccounts[i], nil
		}
	}

	storageaccount := &storageaccounts[0]
	if storageaccount.Tags == nil {
		storageaccount.Tags = map[string]string{}
	}
	storageaccount.Tags["id"] = self.region.Name
	err = self.region.update(jsonutils.Marshal(storageaccount), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Update(%s)", jsonutils.Marshal(storageaccount).String())
	}
	return storageaccount, nil
}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, tmpPath string, callback func(progress float32)) (string, error) {
	reader, sizeBytes, err := image.GetReader(image.ImageId, string(qemuimgfmt.VHD))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	imageNameOnBlob := image.ImageName
	if !strings.HasSuffix(imageNameOnBlob, ".vhd") {
		imageNameOnBlob = fmt.Sprintf("%s.vhd", imageNameOnBlob)
	}
	tmpFile := fmt.Sprintf("%s/%s", tmpPath, imageNameOnBlob)
	defer os.Remove(tmpFile)
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", errors.Wrap(err, "os.Create(tmpFile)")
	}
	defer f.Close()

	// 下载占33%
	r := multicloud.NewProgress(sizeBytes, 33, reader, callback)
	if _, err := io.Copy(f, r); err != nil {
		return "", errors.Wrap(err, "io.Copy(f, reader)")
	}

	storageaccount, err := self.checkStorageAccount()
	if err != nil {
		return "", errors.Wrap(err, "self.checkStorageAccount")
	}

	blobURI, err := storageaccount.UploadFile("image-cache", tmpFile, callback)
	if err != nil {
		return "", errors.Wrap(err, "storageaccount.UploadFile")
	}

	// size, _ := meta.Int("size")

	img, err := self.region.CreateImageByBlob(image.ImageId, image.OsType, blobURI, int32(sizeBytes>>30))
	if err != nil {
		return "", errors.Wrap(err, "CreateImageByBlob")
	}
	if callback != nil {
		callback(100)
	}
	return img.GetGlobalId(), nil
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if image, err := self.region.CreateImage(snapshotId, imageName, osType, imageDesc); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		return image, nil
	}
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(imageId, extId, path)
}

func (self *SStoragecache) downloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	// TODO: need to fix scenarios where image is a public image
	// XXX Qiu Jian
	if image, err := self.region.getPrivateImage(extId); err != nil {
		return nil, err
	} else if snapshotId := image.Properties.StorageProfile.OsDisk.Snapshot.ID; len(snapshotId) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else if uri, err := self.region.GrantAccessSnapshot(snapshotId); err != nil {
		return nil, err
	} else if resp, err := http.Get(uri); err != nil {
		return nil, err
	} else {
		tmpImageFile, err := ioutil.TempFile(path, "temp")
		if err != nil {
			return nil, err
		}
		defer tmpImageFile.Close()
		defer os.Remove(tmpImageFile.Name())
		{
			sf := fileutils.NewSparseFileWriter(tmpImageFile)
			data := make([]byte, DefaultReadBlockSize)
			written := int64(0)
			for {
				n, err := resp.Body.Read(data)
				if n > 0 {
					if n, err := sf.Write(data); err != nil {
						return nil, err
					} else {
						written += int64(n)
					}
				} else if err == io.EOF {
					if written <= resp.ContentLength {
						return nil, fmt.Errorf("got eof: expecting %d bytes, got %d", resp.ContentLength, written)
					}
					break
				} else {
					return nil, err
				}
			}
			if err := sf.PreClose(); err != nil {
				return nil, err
			}
		}

		if _, err := tmpImageFile.Seek(0, os.SEEK_SET); err != nil {
			return nil, errors.Wrap(err, "seek")
		}
		return nil, cloudprovider.ErrNotSupported
	}
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}
