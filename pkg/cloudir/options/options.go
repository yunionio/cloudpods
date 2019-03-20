package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SCloudirOptions struct {
	etcd.SEtcdOptions

	common_options.CommonOptions
}

var (
	Options SCloudirOptions
)
