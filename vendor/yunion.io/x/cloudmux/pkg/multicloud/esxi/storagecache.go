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
	"regexp"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	IMAGE_CACHE_DIR_NAME = "image_cache"
)

type SDatastoreImageCache struct {
	multicloud.SResourceBase
	multicloud.STagBase
	datastore *SDatastore
	host      *SHost
}

type EsxiOptions struct {
	ReasonableCIDREsxi string `help:"Reasonable CIDR in esxi, such as '10.0.0.0/8'" defautl:""`
	TemplateNameRegex  string `help:"Regex of template name"`
	EnableEsxiSwap     bool   `help:"Enable esxi vm swap" default:"false"`
}

var tempalteNameRegex *regexp.Regexp

func InitEsxiConfig(opt EsxiOptions) error {
	var err error
	if len(opt.TemplateNameRegex) != 0 {
		tempalteNameRegex, err = regexp.Compile(opt.TemplateNameRegex)
		if err != nil {
			return errors.Wrap(err, "regexp.Compile")
		}
	}
	return initVMIPV4Filter(opt.ReasonableCIDREsxi)
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

func (self *SDatastoreImageCache) GetPath() string {
	return path.Join(self.datastore.GetMountPoint(), IMAGE_CACHE_DIR_NAME)
}

func (self *SDatastore) getTempalteVMs() ([]*SVirtualMachine, error) {
	return self.FetchTemplateVMs()
}

func (self *SDatastoreImageCache) getTempalteVMs() ([]*SVirtualMachine, error) {
	return self.getTempalteVMs()
}

func (self *SDatastore) getFakeTempateVMs() ([]*SVirtualMachine, error) {
	if tempalteNameRegex == nil {
		return nil, nil
	}
	return self.FetchFakeTempateVMs("")
}

func (self *SDatastoreImageCache) getFakeTempateVMs() ([]*SVirtualMachine, error) {
	return self.datastore.getFakeTempateVMs()
}

func (self *SDatastoreImageCache) GetIImageInImagecache() ([]cloudprovider.ICloudImage, error) {
	ctx := context.Background()
	ret := make([]cloudprovider.ICloudImage, 0, 2)
	files, err := self.datastore.ListDir(ctx, IMAGE_CACHE_DIR_NAME)
	if errors.Cause(err) == errors.ErrNotFound {
		return ret, nil
	}
	if err != nil {
		log.Errorf("GetIImages ListDir fail %s", err)
		return ret, nil
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

var ErrTimeConsuming = errors.Error("time consuming")

func (self *SDatastore) getTemplateVMsFromCache() ([]*SVirtualMachine, error) {
	ihosts, err := self.getCachedAttachedHosts()
	if err != nil {
		return nil, err
	}
	dsRef := self.getDatastore().Self
	ret := make([]*SVirtualMachine, 0)
	for i := range ihosts {
		tvms := ihosts[i].(*SHost).tempalteVMs
		if tvms == nil {
			return nil, ErrTimeConsuming
		}
		for _, vm := range tvms {
			dss := vm.getVirtualMachine().Datastore
			for _, ds := range dss {
				if ds == dsRef {
					ret = append(ret, vm)
					break
				}
			}
		}
	}
	return ret, nil
}

func (self *SDatastoreImageCache) getTemplateVMsFromCache() ([]*SVirtualMachine, error) {
	return self.datastore.getTemplateVMsFromCache()
}

func (self *SDatastore) GetICloudImages(cache *SDatastoreImageCache) ([]cloudprovider.ICloudImage, error) {
	ret := make([]cloudprovider.ICloudImage, 0, 2)

	ids := []string{}
	if self.datacenter.ihosts != nil {
		vms, err := self.getTemplateVMsFromCache()
		if err == nil {
			for i := range vms {
				if !utils.IsInStringArray(vms[i].GetGlobalId(), ids) {
					ret = append(ret, NewVMTemplate(vms[i], cache))
					ids = append(ids, vms[i].GetGlobalId())
				}
			}
			return ret, nil
		}
	}

	realTemplates, err := self.getTempalteVMs()
	if err != nil {
		return nil, errors.Wrap(err, "getTemplateVMs")
	}
	fakeTemplates, err := self.getFakeTempateVMs()
	if err != nil {
		return nil, errors.Wrap(err, "getFakeTempateVMs")
	}

	for i := range realTemplates {
		if !utils.IsInStringArray(realTemplates[i].GetGlobalId(), ids) {
			ret = append(ret, NewVMTemplate(realTemplates[i], cache))
			ids = append(ids, realTemplates[i].GetGlobalId())
		}
	}
	for i := range fakeTemplates {
		if !utils.IsInStringArray(fakeTemplates[i].GetGlobalId(), ids) {
			ret = append(ret, NewVMTemplate(fakeTemplates[i], cache))
			ids = append(ids, fakeTemplates[i].GetGlobalId())
		}
	}
	return ret, nil
}

func (self *SDatastoreImageCache) GetIImageInTemplateVMs() ([]cloudprovider.ICloudImage, error) {
	if self.host == nil {
		return self.datastore.GetICloudImages(self)
	}
	storages, err := self.host.GetIStorages()
	if err != nil {
		return nil, err
	}
	ids := []string{}
	ret := []cloudprovider.ICloudImage{}
	for i := range storages {
		storage := storages[i].(*SDatastore)
		if !storage.isLocalVMFS() {
			continue
		}
		images, err := storage.GetICloudImages(self)
		if err != nil {
			return nil, err
		}
		for i := range images {
			if !utils.IsInStringArray(images[i].GetGlobalId(), ids) {
				ret = append(ret, images[i])
				ids = append(ids, images[i].GetGlobalId())
			}
		}
	}
	return ret, nil
}

func (self *SDatastoreImageCache) GetIImageInTemplateVMsById(id string) (cloudprovider.ICloudImage, error) {
	if tempalteNameRegex != nil {
		vm, err := self.datastore.FetchFakeTempateVMById(id, "")
		if err == nil {
			return NewVMTemplate(vm, self), nil
		}
		if errors.Cause(err) != errors.ErrNotFound {
			return nil, err
		}
	}
	if self.host == nil {
		vm, err := self.datastore.FetchTemplateVMById(id)
		if err == nil {
			return NewVMTemplate(vm, self), nil
		}
		return nil, err
	}
	storages, err := self.host.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		storage := storages[i].(*SDatastore)
		vm, err := storage.FetchTemplateVMById(id)
		if err == nil {
			return NewVMTemplate(vm, self), nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SDatastoreImageCache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastoreImageCache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	images1, err := self.GetIImageInTemplateVMs()
	if err != nil {
		return nil, err
	}
	return images1, nil
	// images2, err := self.GetIImageInImagecache()
	// if err != nil {
	// 	return nil, err
	// }
	// return append(images1, images2...), nil
}

func (self *SDatastoreImageCache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	// check templatevms
	image, err := self.GetIImageInTemplateVMsById(extId)
	if err == nil {
		return image, nil
	}
	if errors.Cause(err) != errors.ErrNotFound {
		return nil, err
	}
	log.Infof("start to GetIImageInImagecache")
	images, err := self.GetIImageInImagecache()
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

func (self *SDatastoreImageCache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
