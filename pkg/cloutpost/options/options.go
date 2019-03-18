package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type CloudSyncOptions struct {
	Provider    string `help:"Public cloud provider" choices:"Aliyun|Azure|Aws|Qcloud"`
	Environment string `help:"environment of public cloud"`
	Cloudregion string `help:"region of public cloud"`
	Zone        string `help:"availability zone of public cloud"`

	etcd.SEtcdOptions

	common_options.CommonOptions
}

var (
	Options CloudSyncOptions
)
