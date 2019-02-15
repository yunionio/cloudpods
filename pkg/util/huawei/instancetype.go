package huawei

import "strconv"

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212656.html
type SInstanceType struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Vcpus        string       `json:"vcpus"`
	RamMB        int          `json:"ram"`            // 内存大小
	OSExtraSpecs OSExtraSpecs `json:"os_extra_specs"` // 扩展规格
}

type OSExtraSpecs struct {
	EcsPerformancetype string `json:"ecs:performancetype"`
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212656.html
func (self *SRegion) fetchInstanceTypes(zoneId string) ([]SInstanceType, error) {
	querys := map[string]string{}
	if len(zoneId) > 0 {
		querys["availability_zone"] = zoneId
	}

	instanceTypes := make([]SInstanceType, 0)
	err := doListAll(self.ecsClient.Flavors.List, querys, &instanceTypes)
	return instanceTypes, err
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, zoneId string) ([]SInstanceType, error) {
	instanceTypes, err := self.fetchInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}

	ret := make([]SInstanceType, 0)
	for _, t := range instanceTypes {
		// cpu & mem & disk都匹配才行
		if t.Vcpus == strconv.Itoa(cpu) && t.RamMB == memMB {
			ret = append(ret, t)
		}
	}

	return ret, nil
}
