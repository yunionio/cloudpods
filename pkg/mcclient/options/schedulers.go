package options

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SchedulerTestBaseOptions struct {
	Mem                 int64    `help:"Memory size (MB), default 512" metavar:"MEMORY" default:"512"`
	Ncpu                int64    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>"`
	Disk                []string `help:"Disk descriptions" nargs:"+"`
	BaremetalDiskConfig []string `help:"Baremetal disk layout configuration"`
	Net                 []string `help:"Network descriptions" metavar:"NETWORK"`
	IsolatedDevice      []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
	SchedTag            []string `help:"Schedule policy, key = SchedTag name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Zone                string   `help:"Preferred zone where virtual server should be created"`
	Host                string   `help:"Preferred host where virtual server should be created"`
	Project             string   `help:"Owner project ID or Name"`
	Hypervisor          string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun"`
	Count               int64    `help:"Create multiple simultaneously, default 1" default:"1"`
	Log                 bool     `help:"Record to schedule history"`
}

func (o SchedulerTestBaseOptions) data(s *mcclient.ClientSession) (*jsonutils.JSONDict, error) {
	data := jsonutils.NewDict()
	if o.Mem > 0 {
		data.Add(jsonutils.NewInt(o.Mem), "vmem_size")
	}
	if o.Ncpu > 0 {
		data.Add(jsonutils.NewInt(o.Ncpu), "vcpu_count")
	}
	for i, d := range o.Disk {
		data.Add(jsonutils.NewString(d), fmt.Sprintf("disk.%d", i))
	}
	for i, n := range o.Net {
		data.Add(jsonutils.NewString(n), fmt.Sprintf("net.%d", i))
	}
	for i, g := range o.IsolatedDevice {
		data.Add(jsonutils.NewString(g), fmt.Sprintf("isolated_device.%d", i))
	}
	if len(o.Host) > 0 {
		data.Add(jsonutils.NewString(o.Host), "prefer_host")
	} else {
		if len(o.Zone) > 0 {
			data.Add(jsonutils.NewString(o.Zone), "prefer_zone")
		}
		if len(o.SchedTag) > 0 {
			for i, aggr := range o.SchedTag {
				data.Add(jsonutils.NewString(aggr), fmt.Sprintf("aggregate.%d", i))
			}
		}
	}
	if len(o.Project) > 0 {
		ret, err := modules.Projects.Get(s, o.Project, nil)
		if err != nil {
			return nil, err
		}
		projectId, err := ret.GetString("id")
		if err != nil {
			return nil, err
		}
		data.Add(jsonutils.NewString(projectId), "owner_tenant_id")
	}
	if len(o.Hypervisor) > 0 {
		data.Add(jsonutils.NewString(o.Hypervisor), "hypervisor")
		if o.Hypervisor == "baremetal" {
			for i, c := range o.BaremetalDiskConfig {
				data.Add(jsonutils.NewString(c), fmt.Sprintf("baremetal_disk_config.%d", i))
			}
		}
	}
	return data, nil
}

func (o SchedulerTestBaseOptions) options() *jsonutils.JSONDict {
	opt := jsonutils.NewDict()
	opt.Add(jsonutils.NewInt(o.Count), "count")
	if o.Log {
		opt.Add(jsonutils.JSONTrue, "record_to_history")
	} else {
		opt.Add(jsonutils.JSONFalse, "record_to_history")
	}
	return opt
}

type SchedulerTestOptions struct {
	SchedulerTestBaseOptions
	SuggestionLimit int64 `help:"Number of schedule candidate informations" default:"50"`
	SuggestionAll   bool  `help:"Show all schedule candidate informations"`
	Details         bool  `help:"Show suggestion details"`
}

func (o SchedulerTestOptions) Params(s *mcclient.ClientSession) (*jsonutils.JSONDict, error) {
	data, err := o.data(s)
	if err != nil {
		return data, err
	}
	params := o.options()
	params.Add(jsonutils.NewInt(o.SuggestionLimit), "suggestion_limit")
	if o.SuggestionAll {
		params.Add(jsonutils.JSONTrue, "suggestion_all")
	} else {
		params.Add(jsonutils.JSONFalse, "suggestion_all")
	}

	if o.Details {
		params.Add(jsonutils.JSONTrue, "suggestion_details")
	} else {
		params.Add(jsonutils.JSONFalse, "suggestion_details")
	}
	params.Add(data, "scheduler")
	return params, nil
}

type SchedulerForecastOptions struct {
	SchedulerTestBaseOptions
}

func (o SchedulerForecastOptions) Params(s *mcclient.ClientSession) (*jsonutils.JSONDict, error) {
	data, err := o.data(s)
	if err != nil {
		return data, err
	}
	params := o.options()
	params.Add(data, "scheduler")
	return params, nil
}
