// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 新建一个通知发送任务
	 */

	type NotificationCreateOptions struct {
		Uid         []string `help:"The user you wanna sent to (Keystone User ID)"`
		CONTACTTYPE string   `help:"User's contacts type" choices:"email|mobile|dingtalk|webconsole"`
		TOPIC       string   `help:"Title or topic of the notification"`
		PRIORITY    string   `help:"Priority of the notification" choices:"normal|important|fatal"`
		MSG         string   `help:"The content of the notification"`
		Remark      string   `help:"Remark or description of the notification"`
		Group       bool     `help:"Send to group"`
	}
	R(&NotificationCreateOptions{}, "notify", "Send a notification to sb", func(s *mcclient.ClientSession, args *NotificationCreateOptions) error {
		msg := notify.SNotifyMessage{}
		if args.Group {
			msg.Gid = args.Uid
		} else {
			msg.Uid = args.Uid
		}

		msg.ContactType = notify.TNotifyChannel(args.CONTACTTYPE)
		msg.Topic = args.TOPIC
		msg.Priority = notify.TNotifyPriority(args.PRIORITY)
		msg.Msg = args.MSG
		msg.Remark = args.Remark

		err := notify.Notifications.Send(s, msg)
		if err != nil {
			return err
		}
		return nil
	})
	/**
	 * 发送全局通知
	 */
	type NotificationBroadcastOptions struct {

		// CONTACTTYPE string `help:"User's contacts type, cloud be email|mobile|dingtalk|/webconsole" choices:"email|mobile|dingtalk|webconsole"`
		Topic    string `required:"true" help:"Title or topic of the notification"`
		Priority string `help:"Priority of the notification" choices:"normal|important|fatal" default:"normal"`
		Msg      string `help:"The content of the notification"`
		Remark   string `help:"Remark or description of the notification"`
		// Group    bool   `help:"Send to group"`
	}

	R(&NotificationBroadcastOptions{}, "notify-broadcast", "Send a notification to all online users", func(s *mcclient.ClientSession, args *NotificationBroadcastOptions) error {
		msg := notify.SNotifyMessage{}
		msg.Broadcast = true
		msg.ContactType = notify.TNotifyChannel("webconsole")
		msg.Topic = args.Topic
		msg.Priority = notify.TNotifyPriority(args.Priority)
		msg.Msg = args.Msg
		msg.Remark = args.Remark

		err := notify.Notifications.Send(s, msg)
		if err != nil {
			return err
		}
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
	R(&NotificationUpdateCallbackOptions{}, "notification-update-callback", "UpdateItem send status of the notification task", func(s *mcclient.ClientSession, args *NotificationUpdateCallbackOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		notification, err := notify.Notifications.Put(s, args.ID, params)
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
		result, err := notify.Notifications.List(s, nil)
		if err != nil {
			return err
		}

		printList(result, notify.Notifications.GetColumns(s))
		return nil
	})

}
