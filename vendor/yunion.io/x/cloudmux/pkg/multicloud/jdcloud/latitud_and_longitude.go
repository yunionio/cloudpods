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

package jdcloud

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-south-1": {Latitude: 23.129110, Longitude: 113.264381, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-north-1": {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-east-2":  {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-east-1":  {Latitude: 33.939763, Longitude: 118.267582, City: api.CITY_SU_QIAN, CountryCode: api.COUNTRY_CODE_CN},
}
