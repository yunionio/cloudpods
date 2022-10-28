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

package hcso

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// China: https://developer.huaweicloud.com/endpoint
// International: https://developer-intl.huaweicloud.com/endpoint
// ref: https://countrycode.org
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-east-2":      {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-east-3":      {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-north-1":     {Latitude: 39.997743, Longitude: 116.304542, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-north-4":     {Latitude: 39.997743, Longitude: 116.304542, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-south-1":     {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-south-2":     {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"ap-southeast-1": {Latitude: 22.396428, Longitude: 114.109497, City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},
	"ap-southeast-2": {Latitude: 13.7563309, Longitude: 100.5017651, City: api.CITY_BANGKOK, CountryCode: api.COUNTRY_CODE_TH},
	"ap-southeast-3": {Latitude: 1.360386, Longitude: 103.821195, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},
	"eu-west-0":      {Latitude: 48.856614, Longitude: 2.3522219, City: api.CITY_PARIS, CountryCode: api.COUNTRY_CODE_FR},
	"cn-northeast-1": {Latitude: 38.91400300000001, Longitude: 121.614682, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-southwest-2": {Latitude: 26.6470035286, Longitude: 106.6302113880, City: api.CITY_GUI_YANG, CountryCode: api.COUNTRY_CODE_CN},
	"af-south-1":     {Latitude: -26.1714537, Longitude: 27.8999389, City: api.CITY_JOHANNESBURG, CountryCode: api.COUNTRY_CODE_ZA},
	"sa-brazil-1":    {Latitude: -23.5505199, Longitude: -46.6333094, City: api.CITY_SAO_PAULO, CountryCode: api.COUNTRY_CODE_BR},
	"na-mexico-1":    {Latitude: 55.1182908, Longitude: 141.0377645, City: api.CITY_MEXICO, CountryCode: api.COUNTRY_CODE_MX},
	"la-south-2":     {Latitude: -33.45206, Longitude: -70.676031, City: api.CITY_SANTIAGO, CountryCode: api.COUNTRY_CODE_CL},
	"cn-north-9":     {Latitude: 41.0178713, Longitude: 113.094978, City: api.CITY_NEI_MENG_GU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-north-219":   {Latitude: 39.997743, Longitude: 116.304542, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
}
