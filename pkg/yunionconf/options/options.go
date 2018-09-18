package options

import "yunion.io/x/onecloud/pkg/cloudcommon"

type YunionConfOptions struct {
	cloudcommon.DBOptions
}

var (
	Options YunionConfOptions
)