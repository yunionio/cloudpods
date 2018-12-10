package modules

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SkusManager struct {
	ResourceManager
}

type ServerSkusManager struct {
	ResourceManager
}

var (
	CloudmetaSkus SkusManager
	ServerSkus    ServerSkusManager
)

func init() {
	CloudmetaSkus = SkusManager{NewCloudmetaManager("sku", "skus",
		[]string{},
		[]string{})}

	ServerSkus = ServerSkusManager{NewComputeManager("serversku", "serverskus",
		[]string{"ID", "Name", "Instance_type_family", "Instance_type_category", "Cpu_core_count",
			"Memory_size_mb", "Os_name", "Sys_disk_resizable", "Sys_disk_type",
			"Sys_disk_min_size_mb", "Sys_disk_max_size_mb", "Attached_disk_type",
			"Attached_disk_size_gb", "Attached_disk_count", "Data_disk_types",
			"Data_disk_max_count", "Nic_max_count", "Cloudregion_id", "Zone_id", "Provider"},
		[]string{})}

	register(&CloudmetaSkus)
	registerCompute(&ServerSkus)
}

func (self *SkusManager) GetSkus(s *mcclient.ClientSession, providerId, regionId, zoneId string, limit, offset int) (*ListResult, error) {
	p := strings.ToLower(providerId)
	r := strings.ToLower(regionId)
	z := strings.ToLower(zoneId)
	url := fmt.Sprintf("/providers/%s/regions/%s/zones/%s/skus?limit=%d&offset=%d", p, r, z, limit, offset)
	ret, err := self._list(s, url, self.KeywordPlural)
	if err != nil {
		return &ListResult{}, err
	}

	return ret, nil
}
