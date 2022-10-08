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
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVMTemplate struct {
	multicloud.SImageBase
	multicloud.STagBase
	cache *SDatastoreImageCache
	vm    *SVirtualMachine
	uuid  string
}

func NewVMTemplate(vm *SVirtualMachine, cache *SDatastoreImageCache) *SVMTemplate {
	return &SVMTemplate{
		cache: cache,
		vm:    vm,
		uuid:  vm.GetGlobalId(),
	}
}

const splitStr = "/"

func toTemplateUuid(templateId string) string {
	ids := strings.Split(templateId, splitStr)
	if len(ids) == 1 {
		return ids[0]
	}
	return ids[1]
}

func toTemplateId(providerId string, templateUuid string) string {
	return fmt.Sprintf("%s%s%s", providerId, splitStr, templateUuid)
}

func (t *SVMTemplate) GetId() string {
	providerId := t.vm.manager.cpcfg.Id
	return toTemplateId(providerId, t.uuid)
}

func (t *SVMTemplate) GetName() string {
	return t.vm.GetName()
}

func (t *SVMTemplate) GetGlobalId() string {
	return t.GetId()
}

func (t *SVMTemplate) GetStatus() string {
	ihosts, err := t.cache.datastore.GetAttachedHosts()
	if err != nil {
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
	for _, ihost := range ihosts {
		host := ihost.(*SHost)
		_, err := host.GetTemplateVMById(t.uuid)
		if err == nil {
			return api.CACHED_IMAGE_STATUS_ACTIVE
		}
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			log.Errorf("fail to find templatevm %q: %v", t.uuid, err)
			return api.CACHED_IMAGE_STATUS_CACHE_FAILED
		}
	}
	return api.CACHED_IMAGE_STATUS_CACHE_FAILED
}

func (t *SVMTemplate) Refresh() error {
	vm, err := t.cache.host.GetTemplateVMById(t.uuid)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return errors.Wrap(err, "no such vm template")
	}
	if err != nil {
		return errors.Wrap(err, "SHost.GetTemplateVMById")
	}
	t.vm = vm
	return nil
}

func (t *SVMTemplate) IsEmulated() bool {
	return false
}

func (t *SVMTemplate) Delete(ctx context.Context) error {
	vm, err := t.cache.host.GetTemplateVMById(t.uuid)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get template vm '%s'", t.uuid)
	}
	return vm.DeleteVM(ctx)
}

func (t *SVMTemplate) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return t.cache
}

func (t *SVMTemplate) GetSizeByte() int64 {
	var sum int
	for i := range t.vm.vdisks {
		vdisk := t.vm.vdisks[i]
		sum += vdisk.GetDiskSizeMB()
	}
	return int64(sum) * (1 << 20)
}

func (t *SVMTemplate) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (t *SVMTemplate) GetImageStatus() string {
	status := t.GetStatus()
	if status == api.CACHED_IMAGE_STATUS_ACTIVE {
		return cloudprovider.IMAGE_STATUS_ACTIVE
	}
	return cloudprovider.IMAGE_STATUS_DELETED
}

func (t *SVMTemplate) GetBios() cloudprovider.TBiosType {
	return t.vm.GetBios()
}

func (t *SVMTemplate) GetOsType() cloudprovider.TOsType {
	return t.vm.GetOsType()
}

func (t *SVMTemplate) GetOsDist() string {
	return t.vm.GetOsDist()
}

func (t *SVMTemplate) GetOsVersion() string {
	return t.vm.GetOsVersion()
}

func (t *SVMTemplate) GetOsLang() string {
	return t.vm.GetOsLang()
}

func (t *SVMTemplate) GetOsArch() string {
	return t.vm.GetOsArch()
}

func (t *SVMTemplate) GetFullOsName() string {
	return t.vm.GetFullOsName()
}

func (t *SVMTemplate) GetMinOsDiskSizeGb() int {
	return int(t.GetSizeByte() / (1 << 30))
}

func (t *SVMTemplate) GetMinRamSizeMb() int {
	return 0
}

func (t *SVMTemplate) GetImageFormat() string {
	return "vmdk"
}

// GetCreateAt return vm's create time by getting the sys disk's create time
func (t *SVMTemplate) GetCreatedAt() time.Time {
	if len(t.vm.vdisks) == 0 {
		return time.Time{}
	}
	return t.vm.vdisks[0].GetCreatedAt()
}

func (t *SVMTemplate) GetSubImages() []cloudprovider.SSubImage {
	subImages := make([]cloudprovider.SSubImage, 0, len(t.vm.vdisks))
	for i := range t.vm.vdisks {
		vdisk := t.vm.vdisks[i]
		sizeMb := vdisk.GetDiskSizeMB()
		subImages = append(subImages, cloudprovider.SSubImage{
			Index:     i,
			SizeBytes: int64(sizeMb) * (1 << 20),
			MinDiskMB: sizeMb,
			MinRamMb:  0,
		})
	}
	return subImages
}
