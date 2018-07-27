package shell

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type SchedulerTestOptions struct {
		Mem                 int64    `help:"Memory size (MB), default 512" metavar:"MEMORY" default:"512"`
		Ncpu                int64    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>"`
		Disk                []string `help:"Disk descriptions" nargs:"+"`
		BaremetalDiskConfig []string `help:"Baremetal disk layout configuration"`
		Net                 []string `help:"Network descriptions" metavar:"NETWORK"`
		IsolatedDevice      []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
		Group               []string `help:"Group of virtual server"`
		SchedTag            []string `help:"Schedule policy, key = SchedTag name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
		Zone                string   `help:"Preferred zone where virtual server should be created"`
		Host                string   `help:"Preferred host where virtual server should be created"`
		Project             string   `help:"Owner project ID or Name"`
		Hypervisor          string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun"`
		Count               int64    `help:"Create multiple simultaneously, default 1" default:"1"`
		Log                 bool     `help:"Record to schedule history"`
		SuggestionLimit     int64    `help:"Number of schedule candidate informations" default:"50"`
		SuggestionAll       bool     `help:"Show all schedule candidate informations"`
		Details             bool     `help:"Show suggestion details"`
	}
	R(&SchedulerTestOptions{}, "scheduler-test", "Emulate schedule process",
		func(s *mcclient.ClientSession, args *SchedulerTestOptions) error {
			params := jsonutils.NewDict()
			data := jsonutils.NewDict()
			params.Add(jsonutils.JSONTrue, "suggestion")
			if args.Mem > 0 {
				data.Add(jsonutils.NewInt(args.Mem), "vmem_size")
			}
			if args.Ncpu > 0 {
				data.Add(jsonutils.NewInt(args.Ncpu), "vcpu_count")
			}
			for i, d := range args.Disk {
				data.Add(jsonutils.NewString(d), fmt.Sprintf("disk.%d", i))
			}
			for i, n := range args.Net {
				data.Add(jsonutils.NewString(n), fmt.Sprintf("net.%d", i))
			}
			for i, g := range args.IsolatedDevice {
				data.Add(jsonutils.NewString(g), fmt.Sprintf("isolated_device.%d", i))
			}
			if len(args.Group) > 0 {
				for i, g := range args.Group {
					data.Add(jsonutils.NewString(g), fmt.Sprintf("group.%d", i))
				}
			}
			if len(args.Host) > 0 {
				data.Add(jsonutils.NewString(args.Host), "prefer_host")
			} else {
				if len(args.Zone) > 0 {
					data.Add(jsonutils.NewString(args.Zone), "prefer_zone")
				}
				if len(args.SchedTag) > 0 {
					for i, aggr := range args.SchedTag {
						data.Add(jsonutils.NewString(aggr), fmt.Sprintf("aggregate.%d", i))
					}
				}
			}
			if len(args.Project) > 0 {
				data.Add(jsonutils.NewString(args.Project), "tenant")
			}
			if len(args.Hypervisor) > 0 {
				data.Add(jsonutils.NewString(args.Hypervisor), "hypervisor")
				if args.Hypervisor == "baremetal" {
					for i, c := range args.BaremetalDiskConfig {
						params.Add(jsonutils.NewString(c), fmt.Sprintf("baremetal_disk_config.%d", i))
					}
				}
			}
			params.Add(jsonutils.NewInt(args.Count), "count")
			if args.Log {
				params.Add(jsonutils.JSONTrue, "record_to_history")
			} else {
				params.Add(jsonutils.JSONFalse, "record_to_history")
			}
			params.Add(jsonutils.NewInt(args.SuggestionLimit), "suggestion_limit")
			if args.SuggestionAll {
				params.Add(jsonutils.JSONTrue, "suggestion_all")
			} else {
				params.Add(jsonutils.JSONFalse, "suggestion_all")
			}

			if args.Details {
				params.Add(jsonutils.JSONTrue, "suggestion_details")
			} else {
				params.Add(jsonutils.JSONFalse, "suggestion_details")
			}
			params.Add(data, "scheduler")
			result, err := modules.SchedManager.Test(s, params)
			if err != nil {
				return err
			}

			listFields := []string{"id", "name", "capacity", "count", "score"}
			if args.Details {
				listFields = append(listFields, "capacity_details", "score_details")
			}

			printList(modules.JSON2ListResult(result), listFields)
			return nil
		})

	type SchedulerCandidateListOptions struct {
		Type   string `help:"Sched type filter" choices:"baremetal|host"`
		Zone   string `help:"Zone ID"`
		Limit  int    `default:"50" help:"Page limit"`
		Offset int    `default:"0" help:"Page offset"`
	}
	R(&SchedulerCandidateListOptions{}, "scheduler-candidate-list", "List scheduler candidates",
		func(s *mcclient.ClientSession, args *SchedulerCandidateListOptions) error {
			params := jsonutils.NewDict()
			if args.Limit > 0 {
				params.Add(jsonutils.NewInt(int64(args.Limit)), "limit")
			}
			if args.Offset > 0 {
				params.Add(jsonutils.NewInt(int64(args.Offset)), "offset")
			}
			if len(args.Zone) > 0 {
				params.Add(jsonutils.NewString(args.Zone), "zone")
			}
			if len(args.Type) > 0 {
				params.Add(jsonutils.NewString(args.Type), "type")
			}
			result, err := modules.SchedManager.CandidateList(s, params)
			if err != nil {
				return err
			}
			printList(modules.JSON2ListResult(result), []string{
				"id", "name", "host_type", "cpu(free/reserverd/total)",
				"mem(free/reserverd/total)", "storage(free/reserverd/total)",
				"status", "host_status", "enable_status"})
			return nil
		})

	type SchedulerCandidateShowOptions struct {
		ID string `help:"ID or name of host"`
	}
	R(&SchedulerCandidateShowOptions{}, "scheduler-candidate-show", "Show candidate detail",
		func(s *mcclient.ClientSession, args *SchedulerCandidateShowOptions) error {
			params := jsonutils.NewDict()
			result, err := modules.SchedManager.CandidateDetail(s, args.ID, params)
			if err != nil {
				return err
			}
			fmt.Println(result.YAMLString())
			return nil
		})

	type SchedulerHistoryListOptions struct {
		Limit  int  `default:"50" help:"Page limit"`
		Offset int  `default:"0" help:"Page offset"`
		All    bool `help:"Show all histories, including scheduler-test"`
	}
	R(&SchedulerHistoryListOptions{}, "scheduler-history-list", "Show scheduler history list",
		func(s *mcclient.ClientSession, args *SchedulerHistoryListOptions) error {
			params := jsonutils.NewDict()
			if args.Limit == 0 {
				params.Add(jsonutils.NewInt(1024), "limit")
			} else {
				params.Add(jsonutils.NewInt(int64(args.Limit)), "limit")
			}
			if args.Offset > 0 {
				params.Add(jsonutils.NewInt(int64(args.Offset)), "offset")
			}
			if args.All {
				params.Add(jsonutils.JSONTrue, "all")
			} else {
				params.Add(jsonutils.JSONFalse, "all")
			}
			result, err := modules.SchedManager.HistoryList(s, params)
			if err != nil {
				return err
			}
			printList(modules.JSON2ListResult(result), []string{
				"session_id", "time", "status", "consuming",
			})
			return nil
		})

	type SchedulerHistoryShowOptions struct {
		ID     string `help:"Session or guest ID"`
		Log    bool   `help:"Show schedule process log"`
		Raw    bool   `help:"Show raw data"`
		Format string `help:"Output format" choices:"json|yaml"`
	}
	R(&SchedulerHistoryShowOptions{}, "scheduler-history-show", "Show scheduler history detail",
		func(s *mcclient.ClientSession, args *SchedulerHistoryShowOptions) error {
			params := jsonutils.NewDict()
			if args.Log {
				params.Add(jsonutils.JSONTrue, "log")
			} else {
				params.Add(jsonutils.JSONFalse, "log")
			}
			if args.Raw {
				params.Add(jsonutils.JSONTrue, "raw")
			} else {
				params.Add(jsonutils.JSONFalse, "false")
			}
			result, err := modules.SchedManager.HistoryShow(s, args.ID, params)
			if err != nil {
				return err
			}
			if args.Format == "json" {
				fmt.Println(result.PrettyString())
			} else {
				fmt.Println(result.YAMLString())
			}
			return nil
		})
}
