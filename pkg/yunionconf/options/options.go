package options

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

type YunionConfOptions struct {
	common_options.CommonOptions
	common_options.DBOptions
}

var (
	Options YunionConfOptions
)
