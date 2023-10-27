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

package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

func (self *SRegion) priceRequest(apiName string, params map[string]interface{}, retval interface{}) error {
	return self.client.invoke("ap-south-1", PRICING_SERVICE_NAME, PRICING_SERVICE_ID, "2017-10-15", apiName, "", params, retval, true)
}

type ProductFilter struct {
	Type  string `json:"Type"`
	Field string `json:"Field"`
	Value string `json:"Value"`
}

func (self *SRegion) GetProducts(serviceCode string, filters []ProductFilter, nextToken string) ([]SInstanceType, string, error) {
	params := map[string]interface{}{
		"ServiceCode": serviceCode,
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}
	if len(filters) > 0 {
		params["Filters"] = filters
	}
	ret := struct {
		FormatVersion string
		NextToken     string `json:"NextToken"`
		PriceList     []string
	}{}
	err := self.priceRequest("GetProducts", params, &ret)
	if err != nil {
		return nil, "", err
	}
	result := []SInstanceType{}
	for _, list := range ret.PriceList {
		obj, err := jsonutils.ParseString(list)
		if err != nil {
			return nil, "", errors.Wrapf(err, "jsonutils.ParseString")
		}
		product := SInstanceType{}
		err = obj.Unmarshal(&product)
		if err != nil {
			return nil, "", errors.Wrapf(err, "Unmarshal")
		}
		result = append(result, product)
	}
	return result, ret.NextToken, nil
}
