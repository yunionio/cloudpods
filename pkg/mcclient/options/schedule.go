package options

import "yunion.io/x/jsonutils"

type ScheduleOptions struct {
	Zone       string   `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Host       string   `help:"Preferred host where virtual server should be created" json:"prefer_host"`
	Schedtag   []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Hypervisor string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun|azure|qcloud|aws"`
}

func (opts *ScheduleOptions) Params() (*jsonutils.JSONDict, error) {
	return optionsStructToParams(opts)
}
