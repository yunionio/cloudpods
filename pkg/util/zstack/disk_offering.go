package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SDiskOffering struct {
	ZStackBasic
	DiskSize          int    `json:"diskSize"`
	Type              string `json:"type"`
	State             string `json:"state"`
	AllocatorStrategy string `json:"allocatorStrategy"`
}

func (region *SRegion) GetDiskOfferings(diskSizeGB int) ([]SDiskOffering, error) {
	offerings := []SDiskOffering{}
	params := []string{}
	if diskSizeGB != 0 {
		params = append(params, "q=diskSize="+fmt.Sprintf("%d", diskSizeGB*1024*1024*1024))
	}
	return offerings, region.client.listAll("disk-offerings", params, &offerings)
}

func (region *SRegion) CreateDiskOffering(diskSizeGB int) (*SDiskOffering, error) {
	params := map[string]interface{}{
		"params": map[string]interface{}{
			"name":     fmt.Sprintf("temp-disk-offering-%dGB", diskSizeGB),
			"diskSize": diskSizeGB * 1024 * 1024 * 1024,
		},
	}
	resp, err := region.client.post("disk-offerings", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	offer := &SDiskOffering{}
	return offer, resp.Unmarshal(offer, "inventory")
}

func (region *SRegion) DeleteDiskOffering(offerId string) error {
	return region.client.delete("disk-offerings", offerId, "")
}
