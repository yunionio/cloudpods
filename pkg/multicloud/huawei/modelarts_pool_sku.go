package huawei

import (
	"strconv"
	"time"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SModelartsPoolSku struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	region *SRegion

	Kind   string                          `json:"kind"`
	Spec   SModelartsResourceflavorsSpec   `json:"spec"`
	Status SModelartsResourceflavorsStatus `json:"status"`
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

type SModelartsResourceflavorsGpuSpec struct {
	Size int    `json:"size"`
	Type string `json:"type"`
}

type SModelartsResourceflavorsStatus struct {
	Phase map[string]interface{} `json:"phase"`
}

type SModelartsResourceflavorsStatusPhase struct {
	CnNorth4a string `json:"cn-north-4a"`
	CnNorth4b string `json:"cn-north-4b"`
	CnNorth4c string `json:"cn-north-4c"`
	CnNorth4g string `json:"cn-north-4g"`
}

func (self *SHuaweiClient) GetIModelartsPoolSku() ([]cloudprovider.ICloudModelartsPoolSku, error) {
	params := make(map[string]interface{})
	resourceflavors := make([]SModelartsPoolSku, 0)
	obj, err := self.modelartsResourceflavors("resourceflavors", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.modelartsResourceflavors")
	}
	obj.Unmarshal(&resourceflavors, "items")
	res := make([]cloudprovider.ICloudModelartsPoolSku, len(resourceflavors))
	for i := 0; i < len(resourceflavors); i++ {
		res[i] = &resourceflavors[i]
	}
	return res, nil
	// return nil, nil
}

func (self *SModelartsPoolSku) GetCreatedAt() time.Time {
	createdAt, _ := time.Parse("2006-01-02T15:04:05CST", time.Now().Format("2006-01-02T15:04:05CST"))
	return createdAt
}

func (self *SModelartsPoolSku) GetGlobalId() string {
	return self.Spec.BillingCode
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
