package cloudprovider

type SGeographicInfo struct {
	Latitude  float32 `list:"user" update:"admin" create:"admin_optional"`
	Longitude float32 `list:"user" update:"admin" create:"admin_optional"`

	City        string `list:"user" width:"32" update:"admin" create:"admin_optional"`
	CountryCode string `list:"user" width:"4" update:"admin" create:"admin_optional"`
}
