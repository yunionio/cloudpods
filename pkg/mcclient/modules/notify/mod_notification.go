package notify

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Notifications NotificationManager
)

type SNotifyMessage struct {
	Uid         string           `json:"uid,omitempty"`
	Gid         string           `json:"uid,omitempty"`
	ContactType []TNotifyChannel `json:"contact_type,omitempty"`
	Topic       string           `json:"topic,omitempty"`
	Priority    TNotifyPriority  `json:"priority,omitempty"`
	Msg         string           `json:"msg,omitempty"`
	Remark      string           `json:"remark,omitempty"`
}

type NotificationManager struct {
	modules.ResourceManager
}

func (manager *NotificationManager) Send(s *mcclient.ClientSession, msg SNotifyMessage) error {
	_, err := manager.Create(s, jsonutils.Marshal(&msg))
	return err
}

func init() {
	Notifications = NotificationManager{
		modules.NewNotifyManager("notification", "notifications",
			[]string{"id", "uid", "contact_type", "topic", "priority", "msg", "received_at", "send_by", "status", "create_at", "update_at", "delete_at", "create_by", "update_by", "delete_by", "is_deleted", "remark"},
			[]string{}),
	}

	modules.Register(&Notifications)
}
