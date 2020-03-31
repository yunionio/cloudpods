package monitor

import (
	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	initAlertNotification()
}

func initAlertNotification() {
	aN := cmdN("alert-notification")
	R(&options.AlertNotificationListOptions{}, aN("list"), "List alert notification pairs",
		func(s *mcclient.ClientSession, args *options.AlertNotificationListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			var result *modulebase.ListResult
			if len(args.Alert) > 0 {
				result, err = monitor.Alertnotification.ListDescendent(s, args.Alert, params)
			} else if len(args.Notification) > 0 {
				result, err = monitor.Alertnotification.ListDescendent2(s, args.Notification, params)
			} else {
				result, err = monitor.Alertnotification.List(s, params)
			}
			if err != nil {
				return err
			}
			printList(result, monitor.Alertnotification.GetColumns(s))
			return nil
		})

	R(&options.AlertNotificationAttachOptions{}, aN("attach"), "Attach a notification to a alert",
		func(s *mcclient.ClientSession, args *options.AlertNotificationAttachOptions) error {
			input := &monitorapi.AlertnotificationCreateInput{
				UsedBy: args.UsedBy,
			}
			ret, err := monitor.Alertnotification.Attach(s, args.ALERT, args.NOTIFICATION, input.JSON(input))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.AlertNotificationAttachOptions{}, aN("detach"), "Detach a notification to a alert",
		func(s *mcclient.ClientSession, args *options.AlertNotificationAttachOptions) error {
			ret, err := monitor.Alertnotification.Detach(s, args.ALERT, args.NOTIFICATION, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
