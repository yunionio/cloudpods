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

package google

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"asia-east1":           api.RegionTaiwan,
	"asia-east2":           api.RegionHongkong,
	"asia-northeast1":      api.RegionTokyo,
	"asia-northeast2":      api.RegionOsaka,
	"asia-northeast3":      api.RegionSeoul,
	"asia-south1":          api.RegionMumbai,
	"asia-south2":          api.RegionDelhi,
	"asia-southeast1":      api.RegionSingapore,
	"asia-southeast2":      api.RegionJakarta,
	"australia-southeast1": api.RegionSydney,
	"australia-southeast2": api.RegionMelbourne,

	"europe-north1":     api.RegionFinland,
	"europe-west1":      api.RegionBelgium,
	"europe-west2":      api.RegionLondon,
	"europe-west3":      api.RegionFrankfurt,
	"europe-west4":      api.RegionHolland,
	"europe-west6":      api.RegionZurich,
	"europe-west8":      api.RegionMilan,
	"europe-west9":      api.RegionParis,
	"europe-central2":   api.RegionWarsaw,
	"europe-southwest1": api.RegionMadrid,
	"europe-west10":     api.RegionDublin,
	"europe-west12":     api.RegionTurin,

	"northamerica-northeast1": api.RegionMontreal,
	"northamerica-northeast2": api.RegionToronto,
	"southamerica-east1":      api.RegionSaoPaulo,
	"us-central1":             api.RegionIowa,
	"us-east1":                api.RegionCarolina,
	"us-east4":                api.RegionNothVirginia,
	"us-east5":                api.RegionColumbus,
	"us-west1":                api.RegionOregon,
	"us-west2":                api.RegionLosAngeles,
	"us-west3":                api.RegionSaltLakeCity,
	"us-west4":                api.RegionLasVegas,
	"us-south1":               api.RegionDallas,
	"southamerica-west1":      api.RegionSantiago,

	"me-west1":    api.RegionColumbus,
	"me-central1": api.RegionDoha,
	"me-central2": api.RegionDamman,
}

var RegionNames = map[string]string{
	"asia-east1":           "台湾",
	"asia-east2":           "香港",
	"asia-northeast1":      "东京",
	"asia-northeast2":      "大阪",
	"asia-northeast3":      "首尔",
	"asia-south1":          "孟买",
	"asia-south2":          "德里",
	"asia-southeast1":      "新加坡",
	"asia-southeast2":      "雅加达",
	"australia-southeast1": "悉尼",
	"australia-southeast2": "墨尔本",

	"europe-north1":     "芬兰",
	"europe-west1":      "比利时",
	"europe-west2":      "伦敦",
	"europe-west3":      "法兰克福",
	"europe-west4":      "荷兰",
	"europe-west6":      "苏黎世",
	"europe-west8":      "米兰",
	"europe-west9":      "巴黎",
	"europe-west10":     "柏林",
	"europe-west12":     "都灵",
	"europe-central2":   "华沙",
	"europe-southwest1": "马德里",

	"northamerica-northeast1": "蒙特利尔",
	"northamerica-northeast2": "多伦多",
	"southamerica-east1":      "圣保罗",
	"southamerica-west1":      "圣地亚哥",
	"us-central1":             "爱荷华",
	"us-east1":                "南卡罗来纳州",
	"us-east4":                "北弗吉尼亚",
	"us-east5":                "哥伦布",
	"us-west1":                "俄勒冈州",
	"us-west2":                "洛杉矶",
	"us-west3":                "盐湖城",
	"us-west4":                "拉斯维加斯",
	"us-south1":               "达拉斯",

	"me-west1":    "特拉维夫",
	"me-central1": "多哈",
	"me-central2": "达曼",

	// Multi-region
	"us":   "美国的多区域",
	"eu":   "欧盟的多区域",
	"asia": "亚洲的多区域",

	// Dual-region
	"nam4": "爱荷华和南卡罗来纳",
	"eur4": "荷兰和芬兰",
}
