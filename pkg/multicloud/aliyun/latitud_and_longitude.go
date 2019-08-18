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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-qingdao":     {Latitude: 36.067108, Longitude: 120.382607, City: api.CITY_QING_DAO, CountryCode: api.COUNTRY_CODE_CN},
	"cn-beijing":     {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-zhangjiakou": {Latitude: 40.767544, Longitude: 114.886337, City: api.CITY_ZHANG_JIA_KOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-huhehaote":   {Latitude: 40.842358, Longitude: 111.749992, City: api.CITY_HU_HE_HAO_TE, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hangzhou":    {Latitude: 30.274084, Longitude: 120.155067, City: api.CITY_HANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-shanghai":    {Latitude: 31.230391, Longitude: 121.473701, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-shenzhen":    {Latitude: 22.543097, Longitude: 114.057861, City: api.CITY_SHEN_ZHEN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hongkong":    {Latitude: 22.396427, Longitude: 114.109497, City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},
	"cn-chengdu":     {Latitude: 30.572815, Longitude: 104.066803, City: api.CITY_CHENG_DU, CountryCode: api.COUNTRY_CODE_CN},
	"ap-northeast-1": {Latitude: 35.709026, Longitude: 139.731995, City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},
	"ap-southeast-1": {Latitude: 1.352083, Longitude: 103.819839, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},
	"ap-southeast-2": {Latitude: -33.868820, Longitude: 151.209290, City: api.CITY_SYDNEY, CountryCode: api.COUNTRY_CODE_AU},
	"ap-southeast-3": {Latitude: 3.139003, Longitude: 101.686852, City: api.CITY_KUALA_LUMPUR, CountryCode: api.COUNTRY_CODE_MY},
	"ap-southeast-5": {Latitude: -6.175110, Longitude: 106.865036, City: api.CITY_JAKARTA, CountryCode: api.COUNTRY_CODE_ID},
	"ap-south-1":     {Latitude: 19.075983, Longitude: 72.877655, City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},
	"us-east-1":      {Latitude: 37.431572, Longitude: -78.656891, City: api.CITY_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},
	"us-west-1":      {Latitude: 37.387474, Longitude: -122.057541, City: api.CITY_SILICONVALLEY, CountryCode: api.COUNTRY_CODE_US},
	"eu-west-1":      {Latitude: 51.507351, Longitude: -0.127758, City: api.CITY_LONDON, CountryCode: api.COUNTRY_CODE_GB},
	"me-east-1":      {Latitude: 25.204849, Longitude: 55.270782, City: api.CITY_DUBAI, CountryCode: api.COUNTRY_CODE_AE},
	"eu-central-1":   {Latitude: 50.110924, Longitude: 8.682127, City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},
}
