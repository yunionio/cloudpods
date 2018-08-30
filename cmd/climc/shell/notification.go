package shell

import (
	//"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 新建一个通知发送任务
	 */
	type NotificationCreateOptions struct {
		UID         string `help:"The user you wanna sent to (Keystone User ID)"`
		CONTACTTYPE string `help:"User's contacts type, maybe email|mobile|dingtalk" choices:"email|mobile|dingtalk"`
		TOPIC       string `help:"Title or topic of the notification"`
		PRIORITY    string `help:"Priority of the notification maybe normal|important|fatal" choices:"normal|important|fatal"`
		MSG         string `help:"The content of the notification"`
		Remark      string `help:"Remark or description of the notification"`
		Group       bool   `help:"Send to group"`
	}
	R(&NotificationCreateOptions{}, "notify", "Send a notification to sb", func(s *mcclient.ClientSession, args *NotificationCreateOptions) error {
		params := jsonutils.NewDict()
		if args.Group {
			params.Add(jsonutils.NewString(args.UID), "gid")
		} else {
			params.Add(jsonutils.NewString(args.UID), "uid")
		}
		params.Add(jsonutils.NewString(args.CONTACTTYPE), "contact_type")
		params.Add(jsonutils.NewString(args.TOPIC), "topic")
		params.Add(jsonutils.NewString(args.PRIORITY), "priority")
		params.Add(jsonutils.NewString(args.MSG), "msg")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		notification, err := modules.Notifications.Create(s, params)
		if err != nil {
			return err
		}
		printObject(notification)
		return nil
	})

	/**
	 * 修改通知发送任务的状态
	 */
	type NotificationUpdateCallbackOptions struct {
		ID     string `help:"ID of the notification send task"`
		STATUS string `help:"Notification send status" choices:"sent_ok|send_fail"`
		Remark string `help:"Remark or description of the operation or fail reason"`
	}
	R(&NotificationUpdateCallbackOptions{}, "notification-update-callback", "Update send status of the notification task", func(s *mcclient.ClientSession, args *NotificationUpdateCallbackOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		notification, err := modules.Notifications.Put(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(notification)
		return nil
	})

	/**
	 * 查询已发送的通知任务
	 */
	type NotificationListOptions struct {
		options.BaseListOptions
	}
	R(&NotificationListOptions{}, "notify-list", "List notification history", func(s *mcclient.ClientSession, args *NotificationListOptions) error {
		result, err := modules.Notifications.List(s, nil)
		if err != nil {
			return err
		}

		printList(result, modules.Notifications.GetColumns(s))
		return nil
	})

}
