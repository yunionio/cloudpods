package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

type SCloudirOptions struct {
	etcd.SEtcdOptions

	cloudcommon.Options
}

var (
	Options SCloudirOptions
)
