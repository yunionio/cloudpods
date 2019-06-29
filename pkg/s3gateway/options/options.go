package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SS3GatewayOptions struct {
	common_options.CommonOptions

	common_options.DBOptions
}

var (
	Options SS3GatewayOptions
)
