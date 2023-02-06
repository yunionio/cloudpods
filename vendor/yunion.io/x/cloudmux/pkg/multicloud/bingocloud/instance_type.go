package bingocloud

import (
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var INSTANCE_CATEGORY_MAP = map[string]string{
	"m1": "M1代",
	"m2": "M2代",
	"m3": "M3代",
	"c1": "C1代",
	"c2": "C2代",
	"c3": "C3代",
	"t1": "T1代",
	"t2": "T2代",
	"t3": "T3代",
}

func getInstanceCategory(family string) string {
	ret, ok := INSTANCE_CATEGORY_MAP[family]
	if ok {
		return ret
	}

	return family
}

type SInstanceType struct {
	cloudprovider.SServerSku

	region *SRegion

	InstanceType string
	DisplayName  string
	Cpu          int
	Ram          int
	Max          int
	Available    int
	Description  string
}

func (self *SInstanceType) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SInstanceType) GetId() string {
	return self.InstanceType
}

func (self *SInstanceType) GetName() string {
	return self.InstanceType
}

func (self *SInstanceType) GetGlobalId() string {
	return self.InstanceType
}

func (self *SInstanceType) GetStatus() string {
	return ""
}

func (self *SInstanceType) Refresh() error {
	return nil
}

func (self *SInstanceType) IsEmulated() bool {
	return false
}

func (self *SInstanceType) GetSysTags() map[string]string {
	return nil
}

func (self *SInstanceType) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SInstanceType) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (self *SInstanceType) GetInstanceTypeFamily() string {
	return strings.Split(self.InstanceType, ".")[0]
}

func (self *SInstanceType) GetInstanceTypeCategory() string {
	return getInstanceCategory(self.GetInstanceTypeFamily())
}

func (self *SInstanceType) GetPrepaidStatus() string {
	return "available"
}

func (self *SInstanceType) GetPostpaidStatus() string {
	return "available"
}

// https://support.huaweicloud.com/productdesc-ecs/ecs_01_0066.html
// https://support.huaweicloud.com/ecs_faq/ecs_faq_0105.html
func (self *SInstanceType) GetCpuArch() string {
	return apis.OS_ARCH_X86
}

func (self *SInstanceType) GetCpuCoreCount() int {
	return self.Cpu
}

func (self *SInstanceType) GetMemorySizeMB() int {
	return self.Ram
}

func (self *SInstanceType) GetOsName() string {
	return ""
}

func (self *SInstanceType) GetSysDiskResizable() bool {
	return false
}

func (self *SInstanceType) GetSysDiskType() string {
	return ""
}

func (self *SInstanceType) GetSysDiskMinSizeGB() int {
	return 0
}

func (self *SInstanceType) GetSysDiskMaxSizeGB() int {
	return 0
}

func (self *SInstanceType) GetAttachedDiskType() string {
	return ""
}

func (self *SInstanceType) GetAttachedDiskSizeGB() int {
	return 0
}

func (self *SInstanceType) GetAttachedDiskCount() int {
	return 0
}

func (self *SInstanceType) GetDataDiskTypes() string {
	return ""
}

func (self *SInstanceType) GetDataDiskMaxCount() int {
	return 0
}

func (self *SInstanceType) GetNicType() string {
	return ""
}

func (self *SInstanceType) GetNicMaxCount() int {
	return 0
}

func (self *SInstanceType) GetGpuAttachable() bool {
	return false
}

func (self *SInstanceType) GetGpuSpec() string {
	return ""
}

func (self *SInstanceType) GetGpuCount() int {
	return 0
}

func (self *SInstanceType) GetGpuMaxCount() int {
	return 0
}

func (self *SInstanceType) Delete() error {
	return nil
}

func (self *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	resp, err := self.invoke("DescribeInstanceTypes", nil)
	if err != nil {
		return nil, err
	}

	result := struct {
		InstanceTypeInfo []SInstanceType
	}{}
	_ = resp.Unmarshal(&result)

	return result.InstanceTypeInfo, err
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	instanceTypes, err := self.GetInstanceTypes()
	if err != nil {
		return nil, err
	}
	iskus := make([]cloudprovider.ICloudSku, len(instanceTypes))
	for i := 0; i < len(instanceTypes); i++ {
		instanceTypes[i].region = self
		iskus[i] = &instanceTypes[i]
	}
	return iskus, nil
}
