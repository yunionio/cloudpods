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

package modules

import (
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SCloudregionManager struct {
	modulebase.ResourceManager
}

var (
	Cloudregions SCloudregionManager
)

type sNameCounter struct {
	Name  string
	Count int
	cloudprovider.SGeographicInfo
}

type tNameCounters []sNameCounter

func (cc tNameCounters) Len() int      { return len(cc) }
func (cc tNameCounters) Swap(i, j int) { cc[i], cc[j] = cc[j], cc[i] }
func (cc tNameCounters) Less(i, j int) bool {
	if cc[i].Count != cc[j].Count {
		return cc[i].Count > cc[j].Count
	}
	return cc[i].Name < cc[j].Name
}

func (this *SCloudregionManager) getRegionAttributeList(session *mcclient.ClientSession, params jsonutils.JSONObject, attr string) (jsonutils.JSONObject, error) {
	paramsDict := params.(*jsonutils.JSONDict)
	if limit, err := paramsDict.Int("limit"); err != nil || limit == 0 {
		paramsDict.Set("limit", jsonutils.NewInt(2048))
	}
	paramsDict.Set("details", jsonutils.JSONFalse)

	listResult, err := this.List(session, params)
	if err != nil {
		return nil, err
	}

	cities := map[string]*sNameCounter{}
	for i := range listResult.Data {
		cityStr, _ := listResult.Data[i].GetString(attr)
		if len(cityStr) == 0 && attr == "city" {
			cityStr = "Other"
		}
		if len(cityStr) > 0 {
			_, ok := cities[cityStr]
			if !ok {
				cities[cityStr] = &sNameCounter{
					Name:  cityStr,
					Count: 0,
				}
				if attr == "city" {
					listResult.Data[i].Unmarshal(&cities[cityStr].SGeographicInfo)
				}
			}
			cities[cityStr].Count += 1
		}
	}

	cityList := make([]sNameCounter, len(cities))
	i := 0
	for k, v := range cities {
		cityList[i] = sNameCounter{Name: k, Count: v.Count, SGeographicInfo: v.SGeographicInfo}
		i += 1
	}

	sort.Sort(tNameCounters(cityList))

	return jsonutils.Marshal(cityList), nil
}

func (this *SCloudregionManager) GetRegionCities(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.getRegionAttributeList(session, params, "city")
}

func (this *SCloudregionManager) GetRegionProviders(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.getRegionAttributeList(session, params, "provider")
}

func (this *SCloudregionManager) GetCityServers(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	objs, err := this.GetRegionCities(session, params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegionCities")
	}
	cities := []sNameCounter{}
	err = objs.Unmarshal(&cities)
	if err != nil {
		return nil, errors.Wrapf(err, "objs.Unmarshal")
	}
	_params := params.(*jsonutils.JSONDict)
	_params.Set("limit", jsonutils.NewInt(1))
	_params.Set("details", jsonutils.NewBool(false))
	for i := range cities {
		_params.Set("city", jsonutils.NewString(cities[i].Name))
		resp, err := Servers.List(session, _params)
		if err != nil {
			return nil, errors.Wrapf(err, "Servers.List")
		}
		cities[i].Count = resp.Total
	}
	sort.Sort(tNameCounters(cities))
	return jsonutils.Marshal(cities), nil
}

func init() {
	Cloudregions = SCloudregionManager{
		NewComputeManager("cloudregion", "cloudregions",
			[]string{"ID", "Name", "Enabled", "Status", "Provider",
				"Latitude", "Longitude", "City", "Country_Code",
				"vpc_count", "zone_count", "guest_count", "guest_increment_count",
				"External_Id"},
			[]string{}),
	}

	registerCompute(&Cloudregions)
}
