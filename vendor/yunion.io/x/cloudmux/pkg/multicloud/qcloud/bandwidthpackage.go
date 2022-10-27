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

package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"
)

type SBandwidthPackageSet struct {
	AddressIp    string
	ResourceId   string
	ResourceType string
}

type SBandwidthPackage struct {
	BandwidthPackageId   string
	BandwidthPackageName string
	ChargeType           string
	CreatedTime          time.Time
	NetworkType          string
	Protocol             string
	ResourceSet          []SBandwidthPackageSet
}

func (region *SRegion) GetBandwidthPackages(ids []string, offset int, limit int) ([]SBandwidthPackage, int, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"Region": region.Region,
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	for idx, id := range ids {
		params[fmt.Sprintf("BandwidthPackageIds.%d", idx)] = id
	}
	packages := []SBandwidthPackage{}
	resp, err := region.vpcRequest("DescribeBandwidthPackages", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeBandwidthPackages")
	}
	err = resp.Unmarshal(&packages, "BandwidthPackageSet")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalCount")
	return packages, int(total), nil
}
