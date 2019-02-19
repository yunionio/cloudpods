package huawei

import "yunion.io/x/onecloud/pkg/cloudprovider"

// China: https://developer.huaweicloud.com/endpoint
// International: https://developer-intl.huaweicloud.com/endpoint
// ref: https://countrycode.org
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-east-2":      {Latitude: 31.210344, Longitude: 121.455364, City: "Shanghai", CountryCode: "CN"},
	"cn-north-1":     {Latitude: 39.997743, Longitude: 116.304542, City: "Beijing", CountryCode: "CN"},
	"cn-south-1":     {Latitude: 23.12911, Longitude: 113.264385, City: "Guangzhou", CountryCode: "CN"},
	"cn-south-2":     {Latitude: 23.12911, Longitude: 113.264385, City: "Guangzhou", CountryCode: "CN"},
	"ap-southeast-1": {Latitude: 22.396428, Longitude: 114.109497, City: "Hongkong", CountryCode: "CN"},
	"ap-southeast-2": {Latitude: 13.7563309, Longitude: 100.5017651, City: "Bangkok", CountryCode: "TH"},
	"eu-west-0":      {Latitude: 48.856614, Longitude: 2.3522219, City: "Paris", CountryCode: "FR"},
	"cn-northeast-1": {Latitude: 38.91400300000001, Longitude: 121.614682, City: "Shanghai", CountryCode: "CN"},
}
