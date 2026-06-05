// Copyright 2023 Yunion
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

package volcengine

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// ref: https://www.volcengine.com/docs/6534/1131814
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 华北
	"cn-beijing":    api.RegionBeijing,
	"cn-beijing2":   api.RegionBeijing,
	"cn-datong":     api.RegionJinzhong,
	"cn-wulanchabu": api.RegionWulanchabu,

	// 华东 / 华南
	"cn-shanghai":  api.RegionShanghai,
	"cn-guangzhou": api.RegionGuangzhou,

	// 港澳台
	"cn-hongkong":     api.RegionHongkong,
	"cn-hongkong-pop": api.RegionHongkong,

	// 亚太
	"ap-southeast-1": api.RegionKualaLumpur,
	"ap-southeast-3": api.RegionJakarta,
}
