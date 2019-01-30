package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

type SLoggerOptions struct {
	cloudcommon.CommonOptions

	cloudcommon.DBOptions
}

var (
	Options SLoggerOptions
)
