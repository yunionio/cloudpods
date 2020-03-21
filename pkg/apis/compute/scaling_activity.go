package compute

import "yunion.io/x/onecloud/pkg/apis"

type ScalingActivityDetails struct {
	apis.StatusStandaloneResourceDetails
	ScalingGroupResourceInfo
	SScalingActivity
}

type ScalingActivityListInput struct {
	apis.StatusStandaloneResourceListInput
	ScalingGroupFilterListInput
}
