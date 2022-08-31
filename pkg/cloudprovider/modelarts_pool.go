package cloudprovider

type ModelartsPoolCreateOption struct {
	Name         string
	PoolDesc     string
	BillingMode  uint
	PeriodType   uint
	PeriodNum    uint
	AutoRenew    uint
	InstanceType string
	NetworkId    string

	WorkType string
}

type Azs struct {
	Az    string `json:"az"`
	Count int    `json:"count"`
}

type ModelartsPoolUpdateOption struct {
	Id       string
	WorkType string
}
