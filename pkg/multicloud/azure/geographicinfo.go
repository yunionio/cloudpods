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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var AzureGeographicInfo = map[string]cloudprovider.SGeographicInfo{
	"southafricanorth":   {City: api.CITY_PRETORIA, CountryCode: api.COUNTRY_CODE_ZA},      //比勒陀利亚, 南非
	"southafricawest":    {City: api.CITY_CAPE_TOWN, CountryCode: api.COUNTRY_CODE_ZA},     //开普敦 南非
	"australiacentral2":  {City: api.CITY_YARRALUMLA, CountryCode: api.COUNTRY_CODE_AU},    //亚拉伦拉 澳大利亚
	"koreasouth":         {City: api.CITY_BUSAN, CountryCode: api.COUNTRY_CODE_KR},         //釜山 韩国
	"canadacentral":      {City: api.CITY_TORONTO, CountryCode: api.COUNTRY_CODE_CA},       //加拿大 多伦多
	"northeurope":        {City: api.CITY_DUBLIN, CountryCode: api.COUNTRY_CODE_IE},        //都柏林 爱尔兰
	"australiacentral":   {City: api.CITY_YARRALUMLA, CountryCode: api.COUNTRY_CODE_AU},    //亚拉伦拉 澳大利亚
	"francecentral":      {City: api.CITY_ALLIER, CountryCode: api.COUNTRY_CODE_FR},        //阿利埃河 法国
	"westus":             {City: api.CITY_SAN_FRANCISCO, CountryCode: api.COUNTRY_CODE_US}, //旧金山 美国
	"japanwest":          {City: api.CITY_OSAKA, CountryCode: api.COUNTRY_CODE_JP},         //大阪市 日本
	"francesouth":        {City: api.CITY_TARN, CountryCode: api.COUNTRY_CODE_FR},          //塔恩 法国
	"eastus":             {City: api.CITY_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},      // 美国 弗吉尼亚
	"westindia":          {City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},        //印度 孟买
	"westcentralus":      {City: api.CITY_UTAH, CountryCode: api.COUNTRY_CODE_US},          //美国 犹他州
	"southeastasia":      {City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},     //新加坡
	"eastasia":           {City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},
	"eastus2":            {City: api.CITY_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},    // 美国 弗吉尼亚
	"japaneast":          {City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},       // 日本 东京
	"ukwest":             {City: api.CITY_HALTON, CountryCode: api.COUNTRY_CODE_GB},      //英国 哈尔顿
	"australiasoutheast": {City: api.CITY_MELBOURNE, CountryCode: api.COUNTRY_CODE_AU},   // 澳大利亚 墨尔本
	"uksouth":            {City: api.CITY_WEST_SUSSEX, CountryCode: api.COUNTRY_CODE_GB}, //英国 西苏塞克斯
	"westus2":            {City: api.CITY_WASHINGTON, CountryCode: api.COUNTRY_CODE_US},  //美国 华盛顿
	"southcentralus":     {City: api.CITY_TEXAS, CountryCode: api.COUNTRY_CODE_US},       //美国 德克萨斯
	"brazilsouth":        {City: api.CITY_SAO_PAULO, CountryCode: api.COUNTRY_CODE_BR},   //巴西 圣保罗
	"koreacentral":       {City: api.CITY_SEOUL, CountryCode: api.COUNTRY_CODE_KR},       //韩国 汉城 -> 首尔
	"centralindia":       {City: api.CITY_MAHARASHTRA, CountryCode: api.COUNTRY_CODE_IN}, //印度 马哈拉施特拉邦
	"northcentralus":     {City: api.CITY_CHICAGO, CountryCode: api.COUNTRY_CODE_US},     //美国 芝加哥
	"centralus":          {City: api.CITY_IOWA, CountryCode: api.COUNTRY_CODE_US},        //美国 爱荷华
	"australiaeast":      {City: api.CITY_SYDNEY, CountryCode: api.COUNTRY_CODE_AU},      //澳大利亚 悉尼
	"westeurope":         {City: api.CITY_HOLLAND, CountryCode: api.COUNTRY_CODE_NL},     //荷兰
	"canadaeast":         {City: api.CITY_QUEBEC, CountryCode: api.COUNTRY_CODE_CA},      //加拿大 魁北克市
	"southindia":         {City: api.CITY_KANCHIPURAM, CountryCode: api.COUNTRY_CODE_IN}, //印度 甘吉布勒姆
	"uaenorth":           {City: api.CITY_DUBAI, CountryCode: api.COUNTRY_CODE_AE},
	"uaecentral":         {City: api.CITY_DUBAI, CountryCode: api.COUNTRY_CODE_AE},
	"switzerlandwest":    {City: api.CITY_GENEVA, CountryCode: api.COUNTRY_CODE_CH},      // 日内瓦
	"switzerlandnorth":   {City: api.CITY_ZURICH, CountryCode: api.COUNTRY_CODE_CH},      // 苏黎世
	"norwaywest":         {City: api.CITY_STAVANGER, CountryCode: api.COUNTRY_CODE_NO},   // 斯塔万格
	"norwayeast":         {City: api.CITY_OSLO, CountryCode: api.COUNTRY_CODE_NO},        // 奥斯陆
	"germanywestcentral": {City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},   // 法兰克福
	"germanynorth":       {City: api.CITY_DELMENHORST, CountryCode: api.COUNTRY_CODE_DE}, // 代尔门霍斯特

	"chinaeast":   {City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"chinaeast2":  {City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"chinanorth":  {City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"chinanorth2": {City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
}
