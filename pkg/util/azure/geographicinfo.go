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

import "yunion.io/x/onecloud/pkg/cloudprovider"

var AzureGeographicInfo = map[string]cloudprovider.SGeographicInfo{
	"southafricanorth":   {City: "Pretoria", CountryCode: "ZA"},      //比勒陀利亚, 南非
	"southafricawest":    {City: "Cape Town", CountryCode: "ZA"},     //开普敦 南非
	"australiacentral2":  {City: "Yarralumla", CountryCode: "AU"},    //亚拉伦拉 澳大利亚
	"koreasouth":         {City: "Busan", CountryCode: "KR"},         //釜山 韩国
	"canadacentral":      {City: "Toronto", CountryCode: "CA"},       //加拿大 多伦多
	"northeurope":        {City: "Dublin", CountryCode: "IE"},        //都柏林 爱尔兰
	"australiacentral":   {City: "Yarralumla", CountryCode: "AU"},    //亚拉伦拉 澳大利亚
	"francecentral":      {City: "Allier", CountryCode: "FR"},        //阿利埃河 法国
	"westus":             {City: "San Francisco", CountryCode: "US"}, //旧金山 美国
	"japanwest":          {City: "Osaka", CountryCode: "JP"},         //大阪市 日本
	"francesouth":        {City: "Tarn", CountryCode: "FR"},          //塔恩 法国
	"eastus":             {City: "Virginia", CountryCode: "US"},      // 美国 弗吉尼亚
	"westindia":          {City: "Mumbai", CountryCode: "IN"},        //印度 孟买
	"westcentralus":      {City: "Utah", CountryCode: "US"},          //美国 犹他州
	"southeastasia":      {City: "Singapore", CountryCode: "SG"},     //新加坡
	"eastasia":           {City: "HongKong", CountryCode: "CN"},
	"eastus2":            {City: "Virginia", CountryCode: "US"},    // 美国 弗吉尼亚
	"japaneast":          {City: "Tokyo", CountryCode: "JP"},       // 日本 东京
	"ukwest":             {City: "Halton", CountryCode: "GB"},      //英国 哈尔顿
	"australiasoutheast": {City: "Melbourne", CountryCode: "AU"},   // 澳大利亚 墨尔本
	"uksouth":            {City: "West Sussex", CountryCode: "GB"}, //英国 西苏塞克斯
	"westus2":            {City: "Washington", CountryCode: "US"},  //美国 华盛顿
	"southcentralus":     {City: "Texas", CountryCode: "US"},       //美国 德克萨斯
	"brazilsouth":        {City: "Sao Paulo", CountryCode: ""},     //巴西 圣保罗
	"koreacentral":       {City: "Seoul", CountryCode: "KR"},       //韩国 汉城
	"centralindia":       {City: "Maharashtra", CountryCode: "IN"}, //印度 马哈拉施特拉邦
	"northcentralus":     {City: "Chicago", CountryCode: "US"},     //美国 芝加哥
	"centralus":          {City: "Iowa", CountryCode: "US"},        //美国 爱荷华
	"australiaeast":      {City: "Sydney", CountryCode: "AU"},      //澳大利亚 悉尼
	"westeurope":         {City: "Holland", CountryCode: "NL"},     //荷兰
	"canadaeast":         {City: "Quebec", CountryCode: "CA"},      //加拿大 魁北克市
	"southindia":         {City: "Kanchipuram", CountryCode: "IN"}, //印度 甘吉布勒姆

	"chinaeast":   {City: "Shanghai", CountryCode: "CN"},
	"chinaeast2":  {City: "Shanghai", CountryCode: "CN"},
	"chinanorth":  {City: "Beijing", CountryCode: "CN"},
	"chinanorth2": {City: "Beijing", CountryCode: "CN"},
}
