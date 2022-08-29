package cloudprovider

type ModelartsPoolCreateOption struct {
	Name           string
	PoolDesc       string
	BillingMode    uint
	PeriodType     uint
	PeriodNum      uint
	AutoRenew      uint
	ResourceFlavor string
	ResourceCount  int
	NetworkId      string
}

type Azs struct {
	Az    string `json:"az"`
	Count int    `json:"count"`
}
