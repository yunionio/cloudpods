package huawei

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// China: https://developer.huaweicloud.com/endpoint
// International: https://developer-intl.huaweicloud.com/endpoint
// ref: https://countrycode.org
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-east-2":      {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
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
}
