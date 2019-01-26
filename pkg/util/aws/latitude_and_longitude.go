package aws

import "yunion.io/x/onecloud/pkg/cloudprovider"

// https://docs.aws.amazon.com/general/latest/gr/rande.html

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"us-east-2":      {Latitude: 40.4172871, Longitude: -82.90712300000001, City: "Ohio", CountryCode: "US"},
	"us-east-1":      {Latitude: 37.4315734, Longitude: -78.6568942, City: "N. Virginia", CountryCode: "US"},
	"us-west-1":      {Latitude: 38.8375215, Longitude: -120.8958242, City: "N. California", CountryCode: "US"},
	"us-west-2":      {Latitude: 43.8041334, Longitude: -120.5542012, City: "Oregon", CountryCode: "US"},
	"ap-south-1":     {Latitude: 19.0759837, Longitude: 72.8776559, City: "Mumbai", CountryCode: "IN"},
	"ap-northeast-3": {Latitude: 34.6937378, Longitude: 135.5021651, City: "Osaka-Local", CountryCode: "JP"},
	"ap-northeast-2": {Latitude: 37.566535, Longitude: 126.9779692, City: "Seoul", CountryCode: "KR"},
	"ap-southeast-1": {Latitude: 1.352083, Longitude: 103.819836, City: "Singapore", CountryCode: "SG"},
	"ap-southeast-2": {Latitude: -33.8688197, Longitude: 151.2092955, City: "Sydney", CountryCode: "AU"},
	"ap-northeast-1": {Latitude: 35.7090259, Longitude: 139.7319925, City: "Tokyo", CountryCode: "JP"},
	"ca-central-1":   {Latitude: 56.130366, Longitude: -106.346771, City: "Central", CountryCode: "CA"},
	"cn-north-1":     {Latitude: 39.90419989999999, Longitude: 116.4073963, City: "Beijing", CountryCode: "CN"},
	"cn-northwest-1": {Latitude: 37.198731, Longitude: 106.1580937, City: "Ningxia", CountryCode: "CN"},
	"eu-central-1":   {Latitude: 50.1109221, Longitude: 8.6821267, City: "Frankfurt", CountryCode: "DE"},
	"eu-west-1":      {Latitude: 53.41291, Longitude: -8.24389, City: "Ireland", CountryCode: "IE"},
	"eu-west-2":      {Latitude: 51.5073509, Longitude: -0.1277583, City: "London", CountryCode: "GB"},
	"eu-west-3":      {Latitude: 48.856614, Longitude: 2.3522219, City: "Paris", CountryCode: "FR"},
	"eu-north-1":     {Latitude: 59.1946, Longitude: 18.47, City: "Stockholm", CountryCode: "SE"},
	"sa-east-1":      {Latitude: -23.5505199, Longitude: -46.63330939999999, City: "San Paulo", CountryCode: "BR"},
	"us-gov-west-1":  {Latitude: 37.09024, Longitude: -95.712891, City: "us-gov-west", CountryCode: "US"},
}
