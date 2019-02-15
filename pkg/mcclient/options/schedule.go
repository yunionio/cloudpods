package options

import "yunion.io/x/jsonutils"

type ScheduleOptions struct {
	Region       string   `help:"Preferred region where virtual server should be created" json:"prefer_region"`
	Zone         string   `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Wire         string   `help:"Preferred wire where virtual server should be created" json:"prefer_wire"`
	Host         string   `help:"Preferred host where virtual server should be created" json:"prefer_host"`
	Schedtag     []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Hypervisor   string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun|azure|qcloud|aws|huawei"`
	ResourceType string   `help:"Resource type" choices:"shared|prepaid|dedicated"`
}

func (opts *ScheduleOptions) Params() (*jsonutils.JSONDict, error) {
	return optionsStructToParams(opts)
}
