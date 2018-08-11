package shell

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出报警日志
	 */
	type AlarmLogListOptions struct {
		BaseListOptions
		Host           string   `help:"Name of the host"`
		Id             int64    `help:"ID of tree node"`
		Label          []string `help:"Labels to this tree node"`
		Metric         string   `help:"Metric name"`
		StartTimeSince string   `help:"Start time since the alarm event"`
		StartTimeUntil string   `help:"Start time until the alarm event"`
		ThisTimeSince  string   `help:"This time since the alarm event"`
		ThisTimeUntil  string   `help:"This time until the alarm event"`
	}
	R(&AlarmLogListOptions{}, "alarmlog-list", "List all alarm's event", func(s *mcclient.ClientSession, args *AlarmLogListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		if len(args.Label) > 0 {
			names := []string{"corp", "owt", "pdl", "srv", "env"}
			segs := make([]string, len(args.Label))

			for i := 0; i < len(args.Label); i += 1 {
				sublabel := args.Label[:i+1]
				pid, _ := modules.TreeNodes.GetNodeIDByLabels(s, sublabel)

				if pid < 0 {
					fmt.Errorf("Invalid node data")
					return nil
				}

				segs[i] = fmt.Sprintf("%s=%d", names[i], pid)
			}

			node_labels := strings.Join(segs, ",")

			params.Add(jsonutils.NewString(node_labels), "node_labels")
		}
		if len(args.Metric) > 0 {
			params.Add(jsonutils.NewString(args.Metric), "metric")
		}
		if len(args.StartTimeSince) > 0 {
			params.Add(jsonutils.NewString(args.StartTimeSince), "start_time_since")
		}
		if len(args.StartTimeUntil) > 0 {
			params.Add(jsonutils.NewString(args.StartTimeUntil), "start_time_until")
		}
		if len(args.ThisTimeSince) > 0 {
			params.Add(jsonutils.NewString(args.ThisTimeSince), "this_time_since")
		}
		if len(args.ThisTimeUntil) > 0 {
			params.Add(jsonutils.NewString(args.ThisTimeUntil), "this_time_until")
		}

		result, err := modules.AlarmLogs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.AlarmLogs.GetColumns(s))
		return nil
	})

}
