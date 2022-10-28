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

package huawei

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// China: https://developer.huaweicloud.com/endpoint
// International: https://developer-intl.huaweicloud.com/endpoint
// ref: https://countrycode.org
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-east-2":      api.RegionShanghai,
	"cn-east-3":      api.RegionShanghai,
	"cn-north-1":     api.RegionBeijing,
	"cn-north-4":     api.RegionBeijing,
	"cn-south-1":     api.RegionGuangzhou,
	"cn-south-2":     api.RegionGuangzhou,
	"ap-southeast-1": api.RegionHongkong,
	"ap-southeast-2": api.RegionBangkok,
	"ap-southeast-3": api.RegionSingapore,
	"eu-west-0":      api.RegionParis,
	"cn-northeast-1": api.RegionDalian,
	"cn-southwest-2": api.RegionGuiyang,
	"af-south-1":     api.RegionJohannesburg,
	"sa-brazil-1":    api.RegionSaoPaulo,
	"na-mexico-1":    api.RegionMexico,
	"la-south-2":     api.RegionSantiago,
	"cn-north-9":     api.RegionNeimenggu,
	"cn-north-219":   api.RegionBeijing,
}
