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

package azure

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var AzureGeographicInfo = map[string]cloudprovider.SGeographicInfo{
	"southafricanorth":   api.RegionPretoria,     //比勒陀利亚, 南非
	"southafricawest":    api.RegionCapeTown,     //开普敦 南非
	"australiacentral2":  api.RegionYarralumla,   //亚拉伦拉 澳大利亚
	"koreasouth":         api.RegionBusan,        //釜山 韩国
	"canadacentral":      api.RegionToronto,      //加拿大 多伦多
	"northeurope":        api.RegionDublin,       //都柏林 爱尔兰
	"australiacentral":   api.RegionYarralumla,   //亚拉伦拉 澳大利亚
	"francecentral":      api.RegionAllier,       //阿利埃河 法国
	"westus":             api.RegionSanFrancisco, //旧金山 美国
	"japanwest":          api.RegionOsaka,        //大阪市 日本
	"francesouth":        api.RegionTarn,         //塔恩 法国
	"eastus":             api.RegionVirginia,     // 美国 弗吉尼亚
	"westindia":          api.RegionMumbai,       //印度 孟买
	"westcentralus":      api.RegionUtah,         //美国 犹他州
	"southeastasia":      api.RegionSingapore,    //新加坡
	"eastasia":           api.RegionHongkong,
	"eastus2":            api.RegionVirginia,    // 美国 弗吉尼亚
	"japaneast":          api.RegionTokyo,       // 日本 东京
	"ukwest":             api.RegionHalton,      //英国 哈尔顿
	"australiasoutheast": api.RegionMelbourne,   // 澳大利亚 墨尔本
	"uksouth":            api.RegionSussex,      //英国 西苏塞克斯
	"westus2":            api.RegionWashington,  //美国 华盛顿
	"southcentralus":     api.RegionTexas,       //美国 德克萨斯
	"brazilsouth":        api.RegionSaoPaulo,    //巴西 圣保罗
	"koreacentral":       api.RegionSeoul,       //韩国 汉城 -> 首尔
	"centralindia":       api.RegionMaharashtra, //印度 马哈拉施特拉邦
	"northcentralus":     api.RegionChicago,     //美国 芝加哥
	"centralus":          api.RegionIowa,        //美国 爱荷华
	"australiaeast":      api.RegionSydney,      //澳大利亚 悉尼
	"westeurope":         api.RegionHolland,     //荷兰
	"canadaeast":         api.RegionQuebec,      //加拿大 魁北克市
	"southindia":         api.RegionKanchipuram, //印度 甘吉布勒姆
	"uaenorth":           api.RegionDubai,
	"uaecentral":         api.RegionDubai,
	"switzerlandwest":    api.RegionGeneva,      // 日内瓦
	"switzerlandnorth":   api.RegionZurich,      // 苏黎世
	"norwaywest":         api.RegionStavanger,   // 斯塔万格
	"norwayeast":         api.RegionOslo,        // 奥斯陆
	"germanywestcentral": api.RegionFrankfurt,   // 法兰克福
	"germanynorth":       api.RegionDelmenhorst, // 代尔门霍斯特
	"westus3":            api.RegionPhoenix,
	"brazilsoutheast":    api.RegionRioDeJaneiro,    // 里约热内卢
	"jioindiawest":       api.RegionJioIndiaWest,    // 贾姆讷格尔
	"jioindiacentral":    api.RegionJioIndiaCentral, // 那格浦尔
	"swedencentral":      api.RegionSandviken,
	"qatarcentral":       api.RegionDoha,
	"polandcentral":      api.RegionWarsaw,
	"italynorth":         api.RegionMilan,
	"israelcentral":      api.RegionTelAviv,
	"brazilus":           api.RegionIndianapolis,

	"chinaeast":   api.RegionShanghai,
	"chinaeast2":  api.RegionShanghai,
	"chinanorth":  api.RegionBeijing,
	"chinanorth2": api.RegionBeijing,
	"chinanorth3": api.RegionZhangjiakou,
}
