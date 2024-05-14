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

package notify

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type NotificationCreateInput struct {
		Receivers   []string `help:"ID or Name of Receiver"`
		Robots      []string `help:"ID or Name of Robot"`
		ContactType string   `help:"Contact type of receiver"`
		TOPIC       string   `help:"Topic"`
		Priority    string   `help:"Priority"`
		MESSAGE     string   `help:"Message"`
		Oldsdk      bool     `help:"Old sdk"`
	}
	R(&NotificationCreateInput{}, "notify-send", "Send a notify message", func(s *mcclient.ClientSession, args *NotificationCreateInput) error {
		var (
			ret jsonutils.JSONObject
			err error
		)
		if args.Oldsdk {
			msg := modules.SNotifyMessage{
				Uid:         args.Receivers,
				Robots:      args.Robots,
				ContactType: modules.TNotifyChannel(args.ContactType),
				Topic:       args.TOPIC,
				Priority:    modules.TNotifyPriority(args.Priority),
				Msg:         args.MESSAGE,
				Remark:      "",
				Broadcast:   false,
			}
			err = modules.Notifications.Send(s, msg)
			if err != nil {
				return err
			}
		} else {
			input := api.NotificationCreateInput{
				Receivers:   args.Receivers,
				Robots:      args.Robots,
				ContactType: args.ContactType,
				Topic:       args.TOPIC,
				Priority:    args.Priority,
				Message:     args.MESSAGE,
			}
			ret, err = modules.Notification.Create(s, jsonutils.Marshal(input))
			if err != nil {
				return err
			}
			printObject(ret)
		}
		return nil
	})
	type NotificationInput struct {
		ID string `help:"Id of notification"`
	}
	R(&NotificationInput{}, "notify-show", "Show a notify message", func(s *mcclient.ClientSession, args *NotificationInput) error {
		ret, err := modules.Notification.Get(s, args.ID, nil)
		if err != nil {
			return nil
		}
		printObject(ret)
		return nil
	})
	type NotificationListInput struct {
		options.BaseListOptions

		ContactType string `help:"contact_type"`
		ReceiverId  string `help:"receiver_id"`
		TopicType   string `help:"topic type"`
	}
	R(&NotificationListInput{}, "notify-list", "List notify message", func(s *mcclient.ClientSession, args *NotificationListInput) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		ret, err := modules.Notification.List(s, params)
		if err != nil {
			return err
		}
		printList(ret, modules.Notification.GetColumns(s))
		return nil
	})
	type NotificationEventInput struct {
		AdvanceDays  int
		Event        string
		Priority     string
		MsgBody      string
		ResourceType string
		Action       string
		Contacts     string
		IsFailed     string
	}
	R(&NotificationEventInput{}, "notify-event-send", "Send notify event message", func(s *mcclient.ClientSession, args *NotificationEventInput) error {
		body, err := jsonutils.ParseString(args.MsgBody)
		if err != nil {
			return err
		}
		dict, ok := body.(*jsonutils.JSONDict)
		if !ok {
			return fmt.Errorf("msg_body should be a json string, like '{'name': 'hello'}'")
		}
		params := api.NotificationManagerEventNotifyInput{
			AdvanceDays:     args.AdvanceDays,
			ReceiverIds:     []string{},
			ResourceDetails: dict,
			Event:           args.Event,
			Priority:        args.Priority,
			ResourceType:    args.ResourceType,
			Action:          api.SAction(args.Action),
			IsFailed:        api.SResult(args.IsFailed),
		}
		_, err = modules.Notification.PerformClassAction(s, "event-notify", jsonutils.Marshal(params))
		if err != nil {
			return fmt.Errorf("unable to EventNotify: %s", err)
		}
		return nil
	})
	type NotificationContactInput struct {
		Subject     string
		Body        string
		ContactType []string
		ReceiverIds []string
		RobotIds    []string
		RoleIds     []string
	}

	R(&NotificationContactInput{}, "notify-contact-send", "Send notify event message", func(s *mcclient.ClientSession, args *NotificationContactInput) error {
		params := api.NotificationManagerContactNotifyInput{
			Subject:      args.Subject,
			Body:         args.Body,
			ReceiverIds:  args.ReceiverIds,
			ContactTypes: args.ContactType,
			RobotIds:     args.RobotIds,
			RoleIds:      args.RoleIds,
		}
		_, err := modules.Notification.PerformClassAction(s, "contact-notify", jsonutils.Marshal(params))
		if err != nil {
			return fmt.Errorf("unable to ContactNotify: %s", err)
		}
		return nil
	})
}
