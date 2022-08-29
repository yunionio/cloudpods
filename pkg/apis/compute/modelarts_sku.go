package compute

import "yunion.io/x/onecloud/pkg/apis"

type RespirceFlavorDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}
