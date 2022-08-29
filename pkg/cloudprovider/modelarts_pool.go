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

	IsTrain    bool
	IsInfer    bool
	IsNotebook bool
}

type Azs struct {
	Az    string `json:"az"`
	Count int    `json:"count"`
}
