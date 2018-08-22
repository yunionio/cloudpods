package mcclient

type IServiceCatalog interface {
	GetServiceURL(service, region, zone, endpointType string) (string, error)
	GetServiceURLs(service, region, zone, endpointType string) ([]string, error)
}
