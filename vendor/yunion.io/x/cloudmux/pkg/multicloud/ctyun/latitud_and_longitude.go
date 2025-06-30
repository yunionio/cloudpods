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

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-beijing-5":   api.RegionBeijing,
	"cn-hangzhou-2":  api.RegionHangzhou,
	"cn-nanjing-4":   api.RegionNanjing,
	"cn-wuhan-3":     api.RegionWuhan,
	"cn-yaan-2":      api.RegionXian,
	"cn-wuhu-1":      api.RegionWuhu,
	"cn-kunming-2":   api.RegionKunming,
	"cn-hefei2":      api.RegionHefei,
	"cn-fujian-3":    api.RegionFujian,
	"cn-jinzhong-2":  api.RegionJinzhong,
	"cn-nanjing-5":   api.RegionNanning,
	"cn-huhehaote-6": api.RegionHuhehaote,
	"cn-nanjing-3":   api.RegionNanjing,
	"cn-haikou-2":    api.RegionHaikou,
	"cn-tianjin-2":   api.RegionTianjin,
	"cn-wuhan-5":     api.RegionWuhan,
	"cn-foshan-3":    api.RegionFoshan,
	"cn-nanning-2":   api.RegionNanning,
	"cn-chenzhou-4":  api.RegionChengzhou,
	"cn-jiangxi-2":   api.RegionJiangxi,
	"cn-hunan-3":     api.RegionChangsha,
	"cn-guangzhou-5": api.RegionGuangzhou,
	"cn-chongqing-2": api.RegionChongqing,
	"cn-shanghai-7":  api.RegionShanghai,
	"cn-shanxi-2":    api.RegionXian,
	"cn-guiyang-1":   api.RegionGuiyang,
	"cn-haerbin-2":   api.RegionHaerbin,
	"cn-zhengzhou-2": api.RegionZhengzhou,
	"cn-xian-4":      api.RegionXian,
	"cn-xian-5":      api.RegionXian,
	"cn-lanzhou-3":   api.RegionLanzhou,
	"cn-nanjing-2":   api.RegionNanjing,
}
