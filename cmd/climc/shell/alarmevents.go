package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	/**
	 * 列出报警事件
	 */
	type AlarmEventListOptions struct {
		BaseListOptions
		NodeLabels     string `help:"Service tree node labels"`
		MetricName     string `help:"Metric name"`
		HostName       string `help:"Host name"`
		HostIp         string `help:"Host IP address"`
		AlarmLevel     string `help:"Alarm level"`
		AlarmCondition string `help:"Concrete alarm rule"`
		Template       string `help:"Template number of the alarm condition"`
		AckStatus      string `help:"Alarm event ack status"`
	}
	R(&AlarmEventListOptions{}, "alarmevent-list", "List all alarm's event", func(s *mcclient.ClientSession, args *AlarmEventListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		if len(args.NodeLabels) > 0 {
			params.Add(jsonutils.NewString(args.NodeLabels), "node_labels")
		}
		if len(args.MetricName) > 0 {
			params.Add(jsonutils.NewString(args.MetricName), "metric_name")
		}
		if len(args.HostName) > 0 {
			params.Add(jsonutils.NewString(args.HostName), "host_name")
		}
		if len(args.HostIp) > 0 {
			params.Add(jsonutils.NewString(args.HostIp), "host_ip")
		}
		if len(args.AlarmLevel) > 0 {
			params.Add(jsonutils.NewString(args.AlarmLevel), "alarm_level")
		}
		if len(args.AlarmCondition) > 0 {
			params.Add(jsonutils.NewString(args.AlarmCondition), "alarm_condition")
		}
		if len(args.Template) > 0 {
			params.Add(jsonutils.NewString(args.Template), "template")
		}
		if len(args.AckStatus) > 0 {
			params.Add(jsonutils.NewString(args.AckStatus), "ack_status")
		}

		result, err := modules.AlarmEvents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.AlarmEvents.GetColumns(s))
		return nil
	})

	/*
	 * 修改报警事件ACK确认状态
	 */
	type AlarmEventUpdateOptions struct {
		ID     string `help:"ID of the alarm event"`
		STATUS int64  `help:"Alarm event ack status" choices:"0|1"`
	}
	R(&AlarmEventUpdateOptions{}, "alarmevent-ack-update", "Update ack status of alarm event", func(s *mcclient.ClientSession, args *AlarmEventUpdateOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.ID), "id")
		tmpObj.Add(jsonutils.NewInt(args.STATUS), "ack_status")
		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(arr, "alarm_events")

		alarmevent, err := modules.AlarmEvents.DoBatchUpdate(s, params)
		if err != nil {
			return err
		}
		printObject(alarmevent)
		return nil
	})

}
