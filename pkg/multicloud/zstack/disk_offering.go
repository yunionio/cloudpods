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

package zstack

import (
	"fmt"
	"net/url"

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
	params := url.Values{}
	if diskSizeGB != 0 {
		params.Add("q", "diskSize="+fmt.Sprintf("%d", diskSizeGB*1024*1024*1024))
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
