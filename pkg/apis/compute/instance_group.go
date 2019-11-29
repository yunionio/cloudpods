package compute

import "yunion.io/x/onecloud/pkg/apis"

type InstanceGroupListInput struct {
	apis.BaseListInput

	// Filter by service type
	ServiceType string `json:"service_type"`
	// Filter by parent id
	ParentId string `json:"parent_id"`
	// Filter by zone id
	ZoneId string `json:"zone_id"`
	// Filter by guest id or name
	Guest string `json:"guest"`
}

type InstanceGroupDetail struct {
	apis.Meta
	SGroup
	GuestCount int64 `json:"guest_count"`
}
