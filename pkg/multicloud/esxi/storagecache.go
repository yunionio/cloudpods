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

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

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

type EsxiOptions struct {
	ReasonableCIDREsxi string `help:"Reasonable CIDR in esxi, such as '10.0.0.0/8'" defautl:""`
	TemplateNameRegex  string `help:"Regex of template name"`
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

func (self *SDatastoreImageCache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDatastoreImageCache) GetPath() string {
	return path.Join(self.datastore.GetMountPoint(), IMAGE_CACHE_DIR_NAME)
}

func (self *SDatastoreImageCache) getTempalteVM() ([]*SVirtualMachine, error) {
	dc := self.datastore.datacenter
	modc := dc.getDatacenter()
	mods := self.datastore.getDatastore()
	templates := make([]mo.VirtualMachine, 0)
	filter := property.Filter{}
	filter["config.template"] = true
	filter["datastore"] = mods.Reference()
	err := self.datastore.manager.scanMObjectsWithFilter(modc.Reference(), VIRTUAL_MACHINE_PROPS, &templates, filter)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjectsWithFilter")
	}
	ret := make([]*SVirtualMachine, 0, len(templates))
	for i := range templates {
		ret = append(ret, NewVirtualMachine(self.datastore.manager, &templates[i], self.datastore.datacenter))
	}
	return ret, nil
}

func (self *SDatastoreImageCache) GetFakeTempateVM(regex string) ([]*SVirtualMachine, error) {
	tNameRegex, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	dc := self.datastore.datacenter
	modc := dc.getDatacenter()
	mods := self.datastore.getDatastore()
	var vms []mo.VirtualMachine
	filter := property.Filter{}
	filter["datastore"] = mods.Reference()
	filter["summary.runtime.powerState"] = types.VirtualMachinePowerStatePoweredOff
	err = self.datastore.manager.scanMObjectsWithFilter(modc.Reference(), []string{"name"}, &vms, filter)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjectsWithFilter")
	}
	objs := make([]types.ManagedObjectReference, 0)
	for i := range vms {
		name := vms[i].Name
		if tNameRegex.MatchString(name) {
			objs = append(objs, vms[i].Reference())
		}
	}
	if len(objs) == 0 {
		return nil, nil
	}
	templates := make([]mo.VirtualMachine, 0, len(objs))
	err = self.datastore.manager.references2Objects(objs, VIRTUAL_MACHINE_PROPS, &templates)
	if err != nil {
		return nil, errors.Wrap(err, "references2Objects")
	}
	ret := make([]*SVirtualMachine, 0, len(templates))
	for i := range templates {
		ret = append(ret, NewVirtualMachine(self.datastore.manager, &templates[i], self.datastore.datacenter))
	}
	return ret, nil
}

func (self *SDatastoreImageCache) getFakeTempateVM() ([]*SVirtualMachine, error) {
	if tempalteNameRegex == nil {
		return []*SVirtualMachine{}, nil
	}
	dc := self.datastore.datacenter
	modc := dc.getDatacenter()
	mods := self.datastore.getDatastore()
	var vms []mo.VirtualMachine
	filter := property.Filter{}
	filter["datastore"] = mods.Reference()
	filter["summary.runtime.powerState"] = types.VirtualMachinePowerStatePoweredOff
	err := self.datastore.manager.scanMObjectsWithFilter(modc.Reference(), []string{"name"}, &vms, filter)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjectsWithFilter")
	}
	objs := make([]types.ManagedObjectReference, 0)
	for i := range vms {
		name := vms[i].Name
		if tempalteNameRegex.MatchString(name) {
			objs = append(objs, vms[i].Reference())
		}
	}
	if len(objs) == 0 {
		return nil, nil
	}
	templates := make([]mo.VirtualMachine, 0, len(objs))
	err = self.datastore.manager.references2Objects(objs, VIRTUAL_MACHINE_PROPS, &templates)
	if err != nil {
		return nil, errors.Wrap(err, "references2Objects")
	}
	ret := make([]*SVirtualMachine, 0, len(templates))
	for i := range templates {
		ret = append(ret, NewVirtualMachine(self.datastore.manager, &templates[i], self.datastore.datacenter))
	}
	return ret, nil
}

func (self *SDatastoreImageCache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	ctx := context.Background()
	ret := make([]cloudprovider.ICloudImage, 0, 2)
	log.Infof("start to GetIImages")

	realTemplates, err := self.getTempalteVM()
	if err != nil {
		return nil, errors.Wrap(err, "getTemplateVM")
	}
	fakeTemplates, err := self.getFakeTempateVM()
	if err != nil {
		return nil, errors.Wrap(err, "getVMWithRegex")
	}

	for i := range realTemplates {
		ret = append(ret, NewVMTemplate(realTemplates[i], self))
	}
	for i := range fakeTemplates {
		ret = append(ret, NewVMTemplate(fakeTemplates[i], self))
	}

	log.Errorf("get templates successfully")
	for i := range fakeTemplates {
		log.Infof("fake template name: %s", fakeTemplates[i].GetName())
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
