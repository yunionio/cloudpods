package aliyun

import "yunion.io/x/onecloud/pkg/cloudprovider"

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-qingdao":     {Latitude: 36.067108, Longitude: 120.382607, City: "Qingdao", CountryCode: "CN"},
	"cn-beijing":     {Latitude: 39.904202, Longitude: 116.407394, City: "Beijing", CountryCode: "CN"},
	"cn-zhangjiakou": {Latitude: 40.767544, Longitude: 114.886337, City: "Zhangjiakou", CountryCode: "CN"},
	"cn-huhehaote":   {Latitude: 40.842358, Longitude: 111.749992, City: "Huhehaote", CountryCode: "CN"},
	"cn-hangzhou":    {Latitude: 30.274084, Longitude: 120.155067, City: "Hangzhou", CountryCode: "CN"},
	"cn-shanghai":    {Latitude: 31.230391, Longitude: 121.473701, City: "Shanghai", CountryCode: "CN"},
	"cn-shenzhen":    {Latitude: 22.543097, Longitude: 114.057861, City: "Shenzhen", CountryCode: "CN"},
	"cn-hongkong":    {Latitude: 22.396427, Longitude: 114.109497, City: "Hongkong", CountryCode: "CN"},
	"ap-northeast-1": {Latitude: 35.709026, Longitude: 139.731995, City: "Tokyo", CountryCode: "JP"},
	"ap-southeast-1": {Latitude: 1.352083, Longitude: 103.819839, City: "Singapore", CountryCode: "SG"},
	"ap-southeast-2": {Latitude: -33.868820, Longitude: 151.209290, City: "Sydney", CountryCode: "AU"},
	"ap-southeast-3": {Latitude: 3.139003, Longitude: 101.686852, City: "Kuala Lumpur", CountryCode: "MY"},
	"ap-southeast-5": {Latitude: -6.175110, Longitude: 106.865036, City: "Jakarta", CountryCode: "ID"},
	"ap-south-1":     {Latitude: 19.075983, Longitude: 72.877655, City: "Mumbai", CountryCode: "IN"},
	"us-east-1":      {Latitude: 37.431572, Longitude: -78.656891, City: "Virgina", CountryCode: "US"},
	"us-west-1":      {Latitude: 37.387474, Longitude: -122.057541, City: "Siliconvalley", CountryCode: "US"},
	"eu-west-1":      {Latitude: 51.507351, Longitude: -0.127758, City: "London", CountryCode: "GB"},
	"me-east-1":      {Latitude: 25.204849, Longitude: 55.270782, City: "Dubai", CountryCode: "AE"},
	"eu-central-1":   {Latitude: 50.110924, Longitude: 8.682127, City: "Frankfurt", CountryCode: "DE"},
}
