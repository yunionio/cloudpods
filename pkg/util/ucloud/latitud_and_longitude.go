package ucloud

import "yunion.io/x/onecloud/pkg/cloudprovider"

// https://docs.ucloud.cn/api/summary/regionlist
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-bj1":       {Latitude: 39.9041999, Longitude: 116.4073963, City: "Beijing", CountryCode: "CN"},
	"cn-bj2":       {Latitude: 39.9041999, Longitude: 116.4073963, City: "Beijing", CountryCode: "CN"},
	"cn-sh":        {Latitude: 31.2303904, Longitude: 121.4737021, City: "Shanghai", CountryCode: "CN"},
	"cn-sh2":       {Latitude: 31.2303904, Longitude: 121.4737021, City: "Shanghai", CountryCode: "CN"},
	"cn-gd":        {Latitude: 23.12911, Longitude: 113.264385, City: "Guangzhou", CountryCode: "CN"},
	"hk":           {Latitude: 22.396428, Longitude: 114.109497, City: "Hong Kong", CountryCode: "CN"},
	"us-ca":        {Latitude: 34.0522342, Longitude: -118.2436849, City: "Los Angeles", CountryCode: "US"},
	"us-ws":        {Latitude: 38.9071923, Longitude: -77.0368707, City: "Washington", CountryCode: "US"},
	"ge-fra":       {Latitude: 50.1109221, Longitude: 8.6821267, City: "Frankfurt", CountryCode: "GE"},
	"th-bkk":       {Latitude: 13.7563309, Longitude: 100.5017651, City: "Bangkok", CountryCode: "TH"},
	"kr-seoul":     {Latitude: 37.566535, Longitude: 126.9779692, City: "Seoul", CountryCode: "KR"},
	"sg":           {Latitude: 1.352083, Longitude: 103.819836, City: "Singapore", CountryCode: "SG"},
	"tw-tp":        {Latitude: 25.0329694, Longitude: 121.5654177, City: "Taipei", CountryCode: "CN"},
	"tw-kh":        {Latitude: 22.6272784, Longitude: 120.3014353, City: "Kaohsiung", CountryCode: "CN"},
	"jpn-tky":      {Latitude: 35.7090259, Longitude: 139.7319925, City: "Tokyo", CountryCode: "JP"},
	"rus-mosc":     {Latitude: 55.755826, Longitude: 37.6172999, City: "Moscow", CountryCode: "RU"},
	"uae-dubai":    {Latitude: 25.2048493, Longitude: 55.2707828, City: "Dubai", CountryCode: "UA"},
	"idn-jakarta":  {Latitude: -6.2087634, Longitude: 106.845599, City: "Jakarta", CountryCode: "ID"},
	"ind-mumbai":   {Latitude: 19.0759837, Longitude: 72.8776559, City: "Mumbai", CountryCode: "IN"},
	"bra-saopaulo": {Latitude: -23.5505199, Longitude: -46.6333094, City: "São Paulo", CountryCode: "BR"},
	"uk-london":    {Latitude: 51.5073509, Longitude: -0.1277583, City: "London", CountryCode: "UK"},
	"afr-nigeria":  {Latitude: 6.5243793, Longitude: 3.3792057, City: "Lagos", CountryCode: "AF"},
	"vn-sng":       {Latitude: 10.8230989, Longitude: 106.6296638, City: "Ho Chi Minh", CountryCode: "VN"},
}
