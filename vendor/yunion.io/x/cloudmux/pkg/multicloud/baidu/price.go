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

package baidu

import (
	"yunion.io/x/pkg/errors"
)

type SInstancePrice struct {
	SpecId     string
	SpecPrices []struct {
		Spec       string
		SpecPrice  float64
		Discount   float64
		TradePrice float64
		Status     string
	}
}

func (region *SRegion) GetInstancePrice(zoneName string, specId, spec string) ([]SInstancePrice, error) {
	params := map[string]interface{}{
		"zoneName":      zoneName,
		"specId":        specId,
		"spec":          spec,
		"paymentTiming": "Postpaid",
	}
	resp, err := region.bccPost("v2/instance/price", nil, params)
	if err != nil {
		return nil, errors.Wrap(err, "get instance price")
	}
	ret := []SInstancePrice{}
	err = resp.Unmarshal(&ret, "price")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal instance price")
	}
	prepaid := []SInstancePrice{}
	params["paymentTiming"] = "Prepaid"
	params["purchaseLength"] = 1
	resp, err = region.bccPost("v2/instance/price", nil, params)
	if err != nil {
		return nil, errors.Wrap(err, "get prepaid instance price")
	}
	err = resp.Unmarshal(&prepaid, "price")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal prepaid price")
	}
	ret = append(ret, prepaid...)
	return ret, nil
}
