package huawei

import (
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SModelartsResourceflavors struct {
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
	Cpu          string                           `json:"cpu"`
	CpuArch      string                           `json:"cpuArch"`
	Gpu          SModelartsResourceflavorsGpuSpec `json:"gpu"`
	Memory       string                           `json:"memory"`
	Type         string                           `json:"type"`
}

type SModelartsResourceflavorsGpuSpec struct {
	Size string `json:"size"`
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

func (self *SRegion) GetResourceflavors() ([]SModelartsResourceflavors, error) {
	params := make(map[string]interface{})
	resourceflavors := make([]SModelartsResourceflavors, 0)
	// err := doListAll(self.ecsClient.Pools.List, params, &pools)

	res, err := self.client.modelartsResourceflavors(self.GetId(), "resourceflavors", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.modelartsResourceflavors")
	}
	res.Unmarshal(&resourceflavors, "items")
	// log.Infoln("this is res", res)
	return resourceflavors, nil
}
