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

package ctyun

import "yunion.io/x/pkg/errors"

type GetPriceOptions struct {
	ResourceType string `default:"VM" choices:"VM|EBS"`
	OnDemand     bool   `default:"true" negative:"non-demand"`
	FlavorName   string
	DiskType     string
}

type SPrice struct {
	TotalPrice     float64
	SubOrderPrices []struct {
		OrderItemPrices []struct {
			ItemId       string
			ResourceType string
			TotalPrice   float64
			FinalPrice   float64
		}
	}
}

func (r *SRegion) GetPrice(opts *GetPriceOptions) (*SPrice, error) {
	params := map[string]interface{}{
		"regionID":     r.RegionId,
		"resourceType": opts.ResourceType,
		"count":        1,
		"onDemand":     opts.OnDemand,
	}
	if !opts.OnDemand {
		params["cycleType"] = "MONTH"
		params["cycleCount"] = 1
	}
	switch opts.ResourceType {
	case "VM":
		params["flavorName"] = opts.FlavorName
		params["sysDiskType"] = "SATA"
		params["sysDiskSize"] = 40
		images, err := r.GetImages("public")
		if err != nil {
			return nil, err
		}
		for _, image := range images {
			if image.OsType == "linux" {
				params["imageUUID"] = image.ImageId
				break
			}
		}
	case "EBS":
		params["diskMode"] = "VBD"
		params["diskSize"] = 10
		params["diskType"] = opts.DiskType
	}
	resp, err := r.post(SERVICE_ECS, "/v4/new-order/query-price", params)
	if err != nil {
		return nil, err
	}
	ret := &SPrice{}
	err = resp.Unmarshal(ret, "returnObj")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}
