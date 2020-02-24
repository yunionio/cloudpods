package compute

import "yunion.io/x/onecloud/pkg/apis"

type ScalingGroupCreateInput struct {
	apis.VirtualResourceCreateInput

	// description: cloud region id or name
	// required: true
	Cloudregion string `json:"cloudregion_id"`

	//swagger: ignore
	CloudregionId string

	// description: hypervisor
	// example: kvm
	Hypervisor string

	// description: min instance number
	// example: 0
	MinInstanceNumber int

	// description: max instance number
	// example: 10
	MaxInstanceNumber int

	// description: desire instance number
	// example: 1
	DesireInstanceNumber int

	// description: guest template id or name
	GuestTemplate string

	// swagger: ignore
	GuestTemplateId string

	// description: expansion principle
	// enum: balanced
	// required: false
	ExpansionPrinciple string

	// description: shrink principle
	// enum: earliest,latest,config_earliest,config_latest
	ShrinkPrinciple string
}

type ScalingGroupListInput struct {
	apis.VirtualResourceListInput

	// description: cloud region
	// example: cr-test
	Cloudregion string `json:"cloudreigon"`

	// description: hypervisor
	// example: kvm
	Hypervisor string `json:"hypervisor"`
}
