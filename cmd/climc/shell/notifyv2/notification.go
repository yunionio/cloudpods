package notifyv2

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type NotificationCreateInput struct {
		Receivers   []string `help:"ID or Name of Receiver"`
		ContactType string   `help:"Contact type of receiver"`
		TOPIC       string   `help:"Topic"`
		Priority    string   `help:"Priority"`
		MESSAGE     string   `help:"Message"`
	}
	R(&NotificationCreateInput{}, "notify-send", "Send a notify message", func(s *mcclient.ClientSession, args *NotificationCreateInput) error {
		input := api.NotificationCreateInput{
			Receivers:   args.Receivers,
			ContactType: args.ContactType,
			Topic:       args.TOPIC,
			Priority:    args.Priority,
			Message:     args.MESSAGE,
		}
		ret, err := modules.Notification.Create(s, jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(ret)
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
}
