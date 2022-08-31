package compute

import "yunion.io/x/onecloud/pkg/apis"

type ModelartsPoolSkuDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}

const (
	MODELARTS_POOL_SKU_AVAILABLE = "available"
	MODELARTS_POOL_SKU_SOLDOUT   = "soldout"
)

type ModelartsPoolSkuListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
}
