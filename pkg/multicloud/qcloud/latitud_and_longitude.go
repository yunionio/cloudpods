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

package qcloud

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"ap-bangkok":        {Latitude: 13.756330, Longitude: 100.501762, City: api.CITY_BANGKOK, CountryCode: api.COUNTRY_CODE_TH},        // 腾讯云 亚太地区(曼谷)
	"ap-beijing":        {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},       // 腾讯云 华北地区(北京)
	"ap-chengdu":        {Latitude: 30.572815, Longitude: 104.066803, City: api.CITY_CHENG_DU, CountryCode: api.COUNTRY_CODE_CN},       // 腾讯云 西南地区(成都)
	"ap-chongqing":      {Latitude: 29.431585, Longitude: 106.912254, City: api.CITY_CHONG_QING, CountryCode: api.COUNTRY_CODE_CN},     // 腾讯云 西南地区(重庆)
	"ap-guangzhou":      {Latitude: 23.129110, Longitude: 113.264381, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},     // 腾讯云 华南地区(广州)
	"ap-guangzhou-open": {Latitude: 23.126593, Longitude: 113.273415, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},     // 腾讯云 华南地区(广州Open)
	"ap-hongkong":       {Latitude: 22.396427, Longitude: 114.109497, City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},      // 腾讯云 东南亚地区(香港)
	"ap-mumbai":         {Latitude: 19.075983, Longitude: 72.877655, City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},          // 腾讯云 亚太地区(孟买)
	"ap-seoul":          {Latitude: 37.566536, Longitude: 126.977966, City: api.CITY_SEOUL, CountryCode: api.COUNTRY_CODE_KR},          // 腾讯云 东南亚地区(首尔)
	"ap-shanghai":       {Latitude: 31.230391, Longitude: 121.473701, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},      // 腾讯云 华东地区(上海)
	"ap-shanghai-fsi":   {Latitude: 31.311033, Longitude: 121.536217, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},      // 腾讯云 华东地区(上海金融)
	"ap-shenzhen-fsi":   {Latitude: 22.531544, Longitude: 114.025467, City: api.CITY_SHEN_ZHEN, CountryCode: api.COUNTRY_CODE_CN},      // 腾讯云 华南地区(深圳金融)
	"ap-singapore":      {Latitude: 1.352083, Longitude: 103.819839, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},       // 腾讯云 东南亚地区(新加坡)
	"ap-tokyo":          {Latitude: 35.709026, Longitude: 139.731995, City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},          // 腾讯云 亚太地区(东京)
	"eu-frankfurt":      {Latitude: 51.165691, Longitude: 10.451526, City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},       // 腾讯云 欧洲地区(德国)
	"eu-moscow":         {Latitude: 55.755825, Longitude: 37.617298, City: api.CITY_MOSCOW, CountryCode: api.COUNTRY_CODE_RU},          // 腾讯云 欧洲地区(莫斯科)
	"na-ashburn":        {Latitude: 37.431572, Longitude: -78.656891, City: api.CITY_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},       // 腾讯云 美国东部(弗吉尼亚)
	"na-siliconvalley":  {Latitude: 37.387474, Longitude: -122.057541, City: api.CITY_SILICONVALLEY, CountryCode: api.COUNTRY_CODE_US}, // 腾讯云 美国西部(硅谷)
	"na-toronto":        {Latitude: 43.653225, Longitude: -79.383186, City: api.CITY_TORONTO, CountryCode: api.COUNTRY_CODE_CA},        // 腾讯云 北美地区(多伦多)

	"ap-nanjing": {Latitude: 32.0584065670, Longitude: 118.7964897811, City: api.CITY_NAN_JING, CountryCode: api.COUNTRY_CODE_CN},
}
