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

package esxi

import (
	"context"
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	IMAGE_CACHE_DIR_NAME = "image_cache"
)

type SDatastoreImageCache struct {
	datastore *SDatastore
	host      *SHost
}

func (self *SDatastoreImageCache) GetId() string {
	if self.host != nil {
		return self.host.GetGlobalId()
	} else {
		return self.datastore.GetGlobalId()
	}
}

func (self *SDatastoreImageCache) GetName() string {
	if self.host != nil {
		return fmt.Sprintf("storage-cache-%s", self.host.GetName())
	} else {
		return fmt.Sprintf("storage-cache-%s", self.datastore.GetName())
	}
}

func (self *SDatastoreImageCache) GetGlobalId() string {
	return self.GetId()
}

func (self *SDatastoreImageCache) GetStatus() string {
	return "available"
}

func (self *SDatastoreImageCache) Refresh() error {
	return nil
}

func (self *SDatastoreImageCache) IsEmulated() bool {
	return false
}

func (self *SDatastoreImageCache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDatastoreImageCache) GetPath() string {
	return path.Join(self.datastore.GetMountPoint(), IMAGE_CACHE_DIR_NAME)
}

func (self *SDatastoreImageCache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	ctx := context.Background()
	ret := make([]cloudprovider.ICloudImage, 0, 2)

	// get vm template with only one disk
	ihosts, err := self.datastore.GetAttachedHosts()
	if err != nil {
		return nil, errors.Wrap(err, "SDatastore.GetAttachedHosts")
	}

	for _, ihost := range ihosts {
		host := ihost.(*SHost)
		tems, err := host.GetTemplateVMs()
		if err != nil {
			log.Errorf("fail to get templateVMs of host '%s' in SDatastoreImageCache.GetIImages", self.host.GetName())
			return ret, nil
		}
		for _, tem := range tems {
			// for now, add vm template with only one disk as cachedimage
			if len(tem.vdisks) != 1 {
				continue
			}
			ret = append(ret, NewVMTemplate(tem, self))
		}
	}

	files, err := self.datastore.ListDir(ctx, IMAGE_CACHE_DIR_NAME)
	if errors.Cause(err) == errors.ErrNotFound {
		return ret, nil
	}
	if err != nil {
		log.Errorf("GetIImages ListDir fail %s", err)
		return nil, err
	}

	validFilenames := make(map[string]bool)

	for i := 0; i < len(files); i += 1 {
		filename := path.Join(IMAGE_CACHE_DIR_NAME, files[i].Name)
		if err := self.datastore.CheckVmdk(ctx, filename); err != nil {
			continue
		}
		vmdkInfo, err := self.datastore.GetVmdkInfo(ctx, filename)
		if err != nil {
			continue
		}
		image := SImage{
			cache:    self,
			filename: filename,
			size:     vmdkInfo.Size(),
			createAt: files[i].Date,
		}
		ret = append(ret, &image)
		vmdkName := files[i].Name
		vmdkExtName := fmt.Sprintf("%s-flat.vmdk", vmdkName[:len(vmdkName)-5])
		validFilenames[vmdkName] = true
		validFilenames[vmdkExtName] = true
	}

	log.Debugf("storage cache contains %#v", validFilenames)
	// cleanup storage cache!!!
	for i := 0; i < len(files); i += 1 {
		if _, ok := validFilenames[files[i].Name]; !ok {
			log.Debugf("delete invalid vmdk file %s!!!", files[i].Name)
			self.datastore.Delete(ctx, path.Join(IMAGE_CACHE_DIR_NAME, files[i].Name))
		}
	}

	return ret, nil
}

func (self *SDatastoreImageCache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	images, err := self.GetIImages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i += 1 {
		if images[i].GetGlobalId() == extId {
			return images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SDatastoreImageCache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastoreImageCache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastoreImageCache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
