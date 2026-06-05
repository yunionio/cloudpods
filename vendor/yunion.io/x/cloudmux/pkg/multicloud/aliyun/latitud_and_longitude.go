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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-qingdao":            api.RegionQingdao,
	"cn-beijing":            api.RegionBeijing,
	"cn-beijing-finance-1":  api.RegionBeijing,
	"cn-zhangjiakou":        api.RegionZhangjiakou,
	"cn-huhehaote":          api.RegionHuhehaote,
	"cn-huhehaote-nebula-1": api.RegionHuhehaote,
	"cn-wulanchabu":         api.RegionWulanchabu,
	"cn-hangzhou":           api.RegionHangzhou,
	"cn-hangzhou-finance":   api.RegionHangzhou,
	"cn-shanghai":           api.RegionShanghai,
	"cn-shanghai-finance-1": api.RegionShanghai,
	"cn-nanjing":            api.RegionNanjing,
	"cn-fuzhou":             api.RegionFujian,
	"cn-fujian":             api.RegionFujian,
	"cn-wuhan":              api.RegionWuhan,
	"cn-wuhan-lr":           api.RegionWuhan,
	"cn-shenzhen":           api.RegionShenzhen,
	"cn-shenzhen-finance-1": api.RegionShenzhen,
	"cn-heyuan":             api.RegionHeyuan,
	"cn-guangzhou":          api.RegionGuangzhou,
	"cn-chengdu":            api.RegionChengdu,
	"cn-zhongwei":           api.RegionNingxia,
	"cn-hongkong":           api.RegionHongkong,
	"cn-north-2-gov-1":      api.RegionBeijing,
	"ap-northeast-1":        api.RegionTokyo,
	"ap-northeast-2":        api.RegionSeoul,
	"ap-southeast-1":        api.RegionSingapore,
	"ap-southeast-2":        api.RegionSydney,
	"ap-southeast-3":        api.RegionKualaLumpur,
	"ap-southeast-5":        api.RegionJakarta,
	"ap-southeast-6":        api.RegionManila,
	"ap-southeast-7":        api.RegionBangkok,
	"ap-southeast-8":        api.RegionKualaLumpur,
	"ap-south-1":            api.RegionMumbai,
	"us-east-1":             api.RegionVirginia,
	"us-west-1":             api.RegionSiliconValley,
	"na-south-1":            api.RegionMexico,
	"eu-west-1":             api.RegionLondon,
	"eu-west-2":             api.RegionParis,
	"eu-central-1":          api.RegionFrankfurt,
	"me-east-1":             api.RegionDubai,
	"me-central-1":          api.RegionDubai,
}
