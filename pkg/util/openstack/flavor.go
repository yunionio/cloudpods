package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SFlavor struct {
	region       *SRegion
	ID           string
	Disk         int
	Ephemeral    int
	ExtraSpecs   ExtraSpecs
	OriginalName string
	Name         string
	RAM          int
	Swap         string
	Vcpus        int8
}

func (region *SRegion) GetFlavors() ([]SFlavor, error) {
	_, resp, err := region.List("compute", "/flavors/detail", "", nil)
	if err != nil {
		return nil, err
	}
	flavors := []SFlavor{}
	return flavors, resp.Unmarshal(&flavors, "flavors")
}

func (region *SRegion) GetFlavor(flavorId string) (*SFlavor, error) {
	_, resp, err := region.Get("compute", "/flavors/"+flavorId, "", nil)
	if err != nil {
		return nil, err
	}
	flavor := &SFlavor{region: region}
	return flavor, resp.Unmarshal(flavor, "flavor")
}

func (region *SRegion) syncFlavor(name string, cpu, memoryMb, diskGB int) (string, error) {
	flavors, err := region.GetFlavors()
	if err != nil {
		return "", err
	}
	if len(name) > 0 {
		for _, flavor := range flavors {
			if flavor.GetName() == name {
				return flavor.ID, nil
			}
		}
	}

	if cpu == 0 && memoryMb == 0 {
		return "", fmt.Errorf("failed to find instance type %s", name)
	}

	for _, flavor := range flavors {
		if flavor.GetCpuCoreCount() == cpu && flavor.GetMemorySizeMB() == memoryMb {
			return flavor.ID, nil
		}
	}

	if len(name) == 0 {
		suffix := ""
		for i := 0; i < 10; i++ {
			switch cpu {
			case 1:
				suffix = "tiny"
			case 2, 3:
				suffix = "small"
			case 4, 6:
				suffix = "medium"
			case 8:
				suffix = "large"
			default:
				suffix = "xlarge"
			}
		}
		for i := 0; i < 10; i++ {
			if _, err := region.GetFlavor(fmt.Sprintf("m%d.%s", i, suffix)); err != nil {
				if err == cloudprovider.ErrNotFound {
					name = fmt.Sprintf("m%d.%s", i, suffix)
					break
				}
			}
		}
		if len(name) == 0 {
			return "", fmt.Errorf("failed to find uniq flavor name for cpu %d memory %d", cpu, memoryMb)
		}
	}

	flavor, err := region.CreateFlavor(name, cpu, memoryMb, diskGB)
	if err != nil {
		return "", err
	}
	return flavor.ID, nil
}

func (region *SRegion) CreateFlavor(name string, cpu int, memoryMb int, diskGB int) (*SFlavor, error) {
	if diskGB < 30 {
		diskGB = 30
	}
	params := map[string]map[string]interface{}{
		"flavor": {
			"name":  name,
			"ram":   memoryMb,
			"vcpus": cpu,
			"disk":  diskGB,
		},
	}
	_, resp, err := region.Post("compute", "/flavors", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	flavor := &SFlavor{}
	return flavor, resp.Unmarshal(flavor, "flavor")
}

func (region *SRegion) DeleteFlavor(flavorId string) error {
	_, err := region.Delete("compute", "/flavors/"+flavorId, "")
	return err
}

func (flavor *SFlavor) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (flavor *SFlavor) IsEmulated() bool {
	return false
}

func (flavor *SFlavor) Refresh() error {
	new, err := flavor.region.GetFlavor(flavor.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(flavor, new)
}

func (flavor *SFlavor) GetName() string {
	if len(flavor.OriginalName) > 0 {
		return flavor.OriginalName
	}
	return flavor.Name
}

func (flavor *SFlavor) GetStatus() string {
	return ""
}

func (flavor *SFlavor) GetId() string {
	return flavor.ID
}

func (flavor *SFlavor) GetGlobalId() string {
	return flavor.ID
}

func (flavor *SFlavor) GetInstanceTypeFamily() string {
	return flavor.GetName()
}

func (flavor *SFlavor) GetInstanceTypeCategory() string {
	return flavor.GetName()
}

func (flavor *SFlavor) GetPrepaidStatus() string {
	return models.SkuStatusSoldout
}

func (flavor *SFlavor) GetPostpaidStatus() string {
	return models.SkuStatusAvailable
}

func (flavor *SFlavor) GetCpuCoreCount() int {
	return int(flavor.Vcpus)
}

func (flavor *SFlavor) GetMemorySizeMB() int {
	return flavor.RAM
}

func (flavor *SFlavor) GetOsName() string {
	return "Any"
}

func (flavor *SFlavor) GetSysDiskResizable() bool {
	return true
}

func (flavor *SFlavor) GetSysDiskType() string {
	return "iscsi"
}

func (flavor *SFlavor) GetSysDiskMinSizeGB() int {
	return 0
}

func (flavor *SFlavor) GetSysDiskMaxSizeGB() int {
	return flavor.Disk
}

func (flavor *SFlavor) GetAttachedDiskType() string {
	return "iscsi"
}

func (flavor *SFlavor) GetAttachedDiskSizeGB() int {
	return 0
}

func (flavor *SFlavor) GetAttachedDiskCount() int {
	return 6
}

func (flavor *SFlavor) GetDataDiskTypes() string {
	return "iscsi"
}

func (flavor *SFlavor) GetDataDiskMaxCount() int {
	return 6
}

func (flavor *SFlavor) GetNicType() string {
	return "vpc"
}

func (flavor *SFlavor) GetNicMaxCount() int {
	return 1
}

func (flavor *SFlavor) GetGpuAttachable() bool {
	return false
}

func (flavor *SFlavor) GetGpuSpec() string {
	return ""
}

func (flavor *SFlavor) GetGpuCount() int {
	return 0
}

func (flavor *SFlavor) GetGpuMaxCount() int {
	return 0
}
