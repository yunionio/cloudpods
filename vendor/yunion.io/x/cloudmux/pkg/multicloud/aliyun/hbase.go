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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

const (
	HBASE_API_VERSION = "2019-01-01"
)

type SHbase struct {
	ClusterId   string
	ClusterName string
}

func (region *SRegion) fetchHbaseEndpoints() error {
	if len(region.client.hbaseEndpoint) > 0 {
		return nil
	}
	client, err := region.getSdkClient()
	if err != nil {
		return err
	}
	endpoint := "hbase.aliyuncs.com"
	resp, err := jsonRequest(client, endpoint, HBASE_API_VERSION, "DescribeRegions", map[string]string{}, region.client.debug)
	if err != nil {
		return err
	}
	ret := []SRegion{}
	err = resp.Unmarshal(&ret, "Regions", "Region")
	if err != nil {
		return err
	}
	hbaseEndpoints := map[string]string{}
	for _, r := range ret {
		hbaseEndpoints[r.RegionId] = r.RegionEndpoint
	}
	region.client.hbaseEndpoint = hbaseEndpoints
	return nil
}

func (region *SRegion) hbaseRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	err := region.fetchHbaseEndpoints()
	if err != nil {
		return nil, errors.Wrapf(err, "fetchHbaseEndpoints")
	}
	client, err := region.getSdkClient()
	if err != nil {
		return nil, err
	}
	endpoint := "hbase.aliyuncs.com"
	if ep, ok := region.client.hbaseEndpoint[region.RegionId]; ok {
		endpoint = ep
	}
	params = region.client.SetResourceGropuId(params)
	return jsonRequest(client, endpoint, HBASE_API_VERSION, apiName, params, region.client.debug)
}

func (self *SRegion) GetHbaseInstances() ([]SHbase, error) {
	ret := []SHbase{}
	pageNumber := 1
	params := map[string]string{
		"RegionId": self.RegionId,
		"PageSize": "100",
	}
	for {
		params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
		resp, err := self.hbaseRequest("DescribeInstances", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			TotalCount int
			Instances  struct {
				Instance []SHbase
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances.Instance...)
		if len(ret) >= part.TotalCount || len(part.Instances.Instance) == 0 {
			break
		}
		pageNumber++
	}
	return ret, nil
}
