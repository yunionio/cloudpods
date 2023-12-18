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

package huawei

import (
	"strconv"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SModelartsPoolSku struct {
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion

	Kind     string                            `json:"kind"`
	Spec     SModelartsResourceflavorsSpec     `json:"spec"`
	Status   SModelartsResourceflavorsStatus   `json:"status"`
	MetaData SModelartsResourceflavorsMetadata `json:"metadata"`
}

type SModelartsResourceflavorsSpec struct {
	BillingCode  string                           `json:"billingCode"`
	BillingModes []int                            `json:"billingMods"`
	Cpu          int                              `json:"cpu"`
	CpuArch      string                           `json:"cpuArch"`
	Gpu          SModelartsResourceflavorsGpuSpec `json:"gpu"`
	Npu          SModelartsResourceflavorsGpuSpec `json:"npu"`
	Memory       string                           `json:"memory"`
	Type         string                           `json:"type"`
}

type SModelartsResourceflavorsMetadata struct {
	Name string `json:"name"`
}

type SModelartsResourceflavorsGpuSpec struct {
	Size int    `json:"size"`
	Type string `json:"type"`
}

type SModelartsResourceflavorsStatus struct {
	Phase map[string]interface{} `json:"phase"`
}

func (self *SRegion) GetIModelartsPoolSku() ([]cloudprovider.ICloudModelartsPoolSku, error) {
	resourceflavors := make([]SModelartsPoolSku, 0)
	obj, err := self.list(SERVICE_MODELARTS, "resourceflavors", nil)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	obj.Unmarshal(&resourceflavors, "items")
	res := make([]cloudprovider.ICloudModelartsPoolSku, len(resourceflavors))
	for i := 0; i < len(resourceflavors); i++ {
		res[i] = &resourceflavors[i]
	}
	return res, nil
}

func (self *SModelartsPoolSku) GetCreatedAt() time.Time {
	createdAt, _ := time.Parse("2006-01-02T15:04:05CST", time.Now().Format("2006-01-02T15:04:05CST"))
	return createdAt
}

func (self *SModelartsPoolSku) GetGlobalId() string {
	return self.MetaData.Name
}

func (self *SModelartsPoolSku) GetId() string {
	return self.Spec.BillingCode
}

func (self *SModelartsPoolSku) GetName() string {
	return self.Spec.BillingCode
}

func (self *SModelartsPoolSku) GetCpuArch() string {
	return self.Spec.CpuArch
}

func (sku *SModelartsPoolSku) GetProcessorType() string {
	if len(sku.GetNpuType()) != 0 {
		return compute.MODELARTS_POOL_SKU_ASCEND
	}
	if len(sku.GetGpuType()) != 0 {
		return compute.MODELARTS_POOL_SKU_GPU
	}
	return compute.MODELARTS_POOL_SKU_CPU
}

func (self *SModelartsPoolSku) GetCpuCoreCount() int {
	return self.Spec.Cpu
}

func (self *SModelartsPoolSku) GetMemorySizeMB() int {
	size, _ := strconv.Atoi(self.Spec.Memory[:len(self.Spec.Memory)-2])
	return size * 1024
}

func (self *SModelartsPoolSku) GetStatus() string {
	for _, v := range self.Status.Phase {
		if v == "normal" {
			return compute.MODELARTS_POOL_SKU_AVAILABLE
		}
	}
	return compute.MODELARTS_POOL_SKU_SOLDOUT
}

func (self *SModelartsPoolSku) GetGpuSize() int {
	return self.Spec.Gpu.Size
}

func (self *SModelartsPoolSku) GetGpuType() string {
	return self.Spec.Gpu.Type
}

func (self *SModelartsPoolSku) GetNpuSize() int {
	return self.Spec.Npu.Size
}

func (self *SModelartsPoolSku) GetNpuType() string {
	return self.Spec.Npu.Type
}

func (self *SModelartsPoolSku) GetPoolType() string {
	return self.Spec.Type
}
