package qcloud

import "yunion.io/x/onecloud/pkg/cloudprovider"

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"ap-bangkok":        {Latitude: 13.756330, Longitude: 100.501762, City: "Bangkok", CountryCode: "TH"},        // 腾讯云 亚太地区(曼谷)
	"ap-beijing":        {Latitude: 39.904202, Longitude: 116.407394, City: "Beijing", CountryCode: "CN"},        // 腾讯云 华北地区(北京)
	"ap-chengdu":        {Latitude: 30.572815, Longitude: 104.066803, City: "Chengdu", CountryCode: "CN"},        // 腾讯云 西南地区(成都)
	"ap-chongqing":      {Latitude: 29.431585, Longitude: 106.912254, City: "Chongqing", CountryCode: "CN"},      // 腾讯云 西南地区(重庆)
	"ap-guangzhou":      {Latitude: 23.129110, Longitude: 113.264381, City: "Guangzhou", CountryCode: "CN"},      // 腾讯云 华南地区(广州)
	"ap-guangzhou-open": {Latitude: 23.126593, Longitude: 113.273415, City: "Guangzhou", CountryCode: "CN"},      // 腾讯云 华南地区(广州Open)
	"ap-hongkong":       {Latitude: 22.396427, Longitude: 114.109497, City: "Hongkong", CountryCode: "HK"},       // 腾讯云 东南亚地区(香港)
	"ap-mumbai":         {Latitude: 19.075983, Longitude: 72.877655, City: "Mumbai", CountryCode: "IN"},          // 腾讯云 亚太地区(孟买)
	"ap-seoul":          {Latitude: 37.566536, Longitude: 126.977966, City: "Seoul", CountryCode: "KR"},          // 腾讯云 东南亚地区(首尔)
	"ap-shanghai":       {Latitude: 31.230391, Longitude: 121.473701, City: "Shanghai", CountryCode: "CN"},       // 腾讯云 华东地区(上海)
	"ap-shanghai-fsi":   {Latitude: 31.311033, Longitude: 121.536217, City: "Shanghai", CountryCode: "CN"},       // 腾讯云 华东地区(上海金融)
	"ap-shenzhen-fsi":   {Latitude: 22.531544, Longitude: 114.025467, City: "Shenzhen", CountryCode: "CN"},       // 腾讯云 华南地区(深圳金融)
	"ap-singapore":      {Latitude: 1.352083, Longitude: 103.819839, City: "Singapore", CountryCode: "SG"},       // 腾讯云 东南亚地区(新加坡)
	"ap-tokyo":          {Latitude: 35.709026, Longitude: 139.731995, City: "Tokyo", CountryCode: "JP"},          // 腾讯云 亚太地区(东京)
	"eu-frankfurt":      {Latitude: 51.165691, Longitude: 10.451526, City: "Frankfurt", CountryCode: "DE"},       // 腾讯云 欧洲地区(德国)
	"eu-moscow":         {Latitude: 55.755825, Longitude: 37.617298, City: "Moscow", CountryCode: "RU"},          // 腾讯云 欧洲地区(莫斯科)
	"na-ashburn":        {Latitude: 37.431572, Longitude: -78.656891, City: "Virgina", CountryCode: "US"},        // 腾讯云 美国东部(弗吉尼亚)
	"na-siliconvalley":  {Latitude: 37.387474, Longitude: -122.057541, City: "Siliconvalley", CountryCode: "US"}, // 腾讯云 美国西部(硅谷)
	"na-toronto":        {Latitude: 43.653225, Longitude: -79.383186, City: "Toronto", CountryCode: "CA"},        // 腾讯云 北美地区(多伦多)
}
