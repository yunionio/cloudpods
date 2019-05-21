package cloudprovider

type SEip struct {
	Name              string
	BandwidthMbps     int
	ChargeType        string
	BGPType           string
	NetworkExternalId string
	IP                string
}
