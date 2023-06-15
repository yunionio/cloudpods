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

func (insType *SInstanceType) GetCreatedAt() time.Time {
	return time.Time{}
}

func (insType *SInstanceType) GetId() string {
	return insType.InstanceType
}

func (insType *SInstanceType) GetName() string {
	return insType.InstanceType
}

func (insType *SInstanceType) GetDescription() string {
	return ""
}

func (insType *SInstanceType) GetGlobalId() string {
	return insType.InstanceType
}

func (insType *SInstanceType) GetStatus() string {
	return ""
}

func (insType *SInstanceType) Refresh() error {
	return nil
}

func (insType *SInstanceType) IsEmulated() bool {
	return false
}

func (insType *SInstanceType) GetSysTags() map[string]string {
	return nil
}

func (insType *SInstanceType) GetTags() (map[string]string, error) {
	return nil, nil
}

func (insType *SInstanceType) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (insType *SInstanceType) GetInstanceTypeFamily() string {
	return strings.Split(insType.InstanceType, ".")[0]
}

func (insType *SInstanceType) GetInstanceTypeCategory() string {
	return getInstanceCategory(insType.GetInstanceTypeFamily())
}

func (insType *SInstanceType) GetPrepaidStatus() string {
	return "available"
}

func (insType *SInstanceType) GetPostpaidStatus() string {
	return "available"
}

// https://support.huaweicloud.com/productdesc-ecs/ecs_01_0066.html
// https://support.huaweicloud.com/ecs_faq/ecs_faq_0105.html
func (insType *SInstanceType) GetCpuArch() string {
	return apis.OS_ARCH_X86
}

func (insType *SInstanceType) GetCpuCoreCount() int {
	return insType.Cpu
}

func (insType *SInstanceType) GetMemorySizeMB() int {
	return insType.Ram
}

func (insType *SInstanceType) GetOsName() string {
	return ""
}

func (insType *SInstanceType) GetSysDiskResizable() bool {
	return false
}

func (insType *SInstanceType) GetSysDiskType() string {
	return ""
}

func (insType *SInstanceType) GetSysDiskMinSizeGB() int {
	return 0
}

func (insType *SInstanceType) GetSysDiskMaxSizeGB() int {
	return 0
}

func (insType *SInstanceType) GetAttachedDiskType() string {
	return ""
}

func (insType *SInstanceType) GetAttachedDiskSizeGB() int {
	return 0
}

func (insType *SInstanceType) GetAttachedDiskCount() int {
	return 0
}

func (insType *SInstanceType) GetDataDiskTypes() string {
	return ""
}

func (insType *SInstanceType) GetDataDiskMaxCount() int {
	return 0
}

func (insType *SInstanceType) GetNicType() string {
	return ""
}

func (insType *SInstanceType) GetNicMaxCount() int {
	return 0
}

func (insType *SInstanceType) GetGpuAttachable() bool {
	return false
}

func (insType *SInstanceType) GetGpuSpec() string {
	return ""
}

func (insType *SInstanceType) GetGpuCount() string {
	return ""
}

func (insType *SInstanceType) GetGpuMaxCount() int {
	return 0
}

func (insType *SInstanceType) Delete() error {
	return nil
}

func (insType *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	resp, err := insType.invoke("DescribeInstanceTypes", nil)
	if err != nil {
		return nil, err
	}

	result := struct {
		InstanceTypeInfo []SInstanceType
	}{}
	_ = resp.Unmarshal(&result)

	return result.InstanceTypeInfo, err
}

func (insType *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	instanceTypes, err := insType.GetInstanceTypes()
	if err != nil {
		return nil, err
	}
	iskus := make([]cloudprovider.ICloudSku, len(instanceTypes))
	for i := 0; i < len(instanceTypes); i++ {
		instanceTypes[i].region = insType
		iskus[i] = &instanceTypes[i]
	}
	return iskus, nil
}
