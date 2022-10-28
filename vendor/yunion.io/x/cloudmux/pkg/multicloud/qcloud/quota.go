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
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type ImageQuota struct {
	region        *SRegion
	ImageNumQuota int
}

func (iq *ImageQuota) GetGlobalId() string {
	return "ImageNumQuota"
}

func (iq *ImageQuota) GetName() string {
	return "ImageNumQuota"
}

func (iq *ImageQuota) GetDesc() string {
	return ""
}

func (iq *ImageQuota) GetQuotaType() string {
	return "ImageNumQuota"
}

func (iq *ImageQuota) GetMaxQuotaCount() int {
	return iq.ImageNumQuota
}

func (iq *ImageQuota) GetCurrentQuotaUsedCount() int {
	_, count, err := iq.region.GetImages("", "PRIVATE_IMAGE", nil, "", 0, iq.ImageNumQuota)
	if err != nil {
		log.Errorf("get private image error: %v", err)
		return -1
	}
	return count
}

func (region *SRegion) GetImageQuota() (*ImageQuota, error) {
	params := map[string]string{}
	resp, err := region.cvmRequest("DescribeImageQuota", params, true)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeImageQuota")
	}
	imageQuota := &ImageQuota{region: region}
	return imageQuota, resp.Unmarshal(imageQuota)
}

type QuotaSet struct {
	QuotaId      string
	QuotaCurrent int
	QuotaLimit   int
}

func (qs *QuotaSet) GetGlobalId() string {
	return strings.ToLower(qs.QuotaId)
}

func (qs *QuotaSet) GetName() string {
	return qs.QuotaId
}

func (qs *QuotaSet) GetDesc() string {
	return ""
}

func (qs *QuotaSet) GetQuotaType() string {
	return qs.QuotaId
}

func (qs *QuotaSet) GetMaxQuotaCount() int {
	return qs.QuotaLimit
}

func (qs *QuotaSet) GetCurrentQuotaUsedCount() int {
	return qs.QuotaCurrent
}

func (region *SRegion) GetQuota(action string) ([]QuotaSet, error) {
	params := map[string]string{}
	resp, err := region.vpcRequest(action, params)
	if err != nil {
		return nil, errors.Wrap(err, action)
	}
	quotas := []QuotaSet{}
	err = resp.Unmarshal(&quotas, "QuotaSet")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return quotas, nil
}

func (region *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	ret := []cloudprovider.ICloudQuota{}
	imageQ, err := region.GetImageQuota()
	if err != nil {
		return nil, errors.Wrap(err, "GetImageQuota")
	}
	ret = append(ret, imageQ)
	for _, action := range []string{"DescribeAddressQuota", "DescribeBandwidthPackageQuota"} {
		quotas, err := region.GetQuota(action)
		if err != nil {
			return nil, errors.Wrapf(err, "GetQuota(%s)", action)
		}
		for i := range quotas {
			ret = append(ret, &quotas[i])
		}
	}
	return ret, nil
}
