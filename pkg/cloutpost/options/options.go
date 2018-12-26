package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

type CloudSyncOptions struct {
	Provider    string `help:"Public cloud provider" choices:"Aliyun|Azure|Aws|Qcloud"`
	Environment string `help:"environment of public cloud"`
	Cloudregion string `help:"region of public cloud"`
	Zone        string `help:"availability zone of public cloud"`

	etcd.SEtcdOptions

	cloudcommon.Options
}

var (
	Options CloudSyncOptions
)
