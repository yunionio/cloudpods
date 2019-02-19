package notify

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestNotificationManager(t *testing.T) {
	msg := SNotifyMessage{
		Uid: "testuser",
		ContactType: []TNotifyChannel{
			NotifyByEmail, NotifyByWebConsole,
		},
		Topic:    "test message",
		Priority: NotifyPriorityNormal,
		Msg:      "This is a test message. Yey!!",
		Remark:   "Yunion",
	}
	msgJson := jsonutils.Marshal(msg)
	t.Logf("msg: %s", msgJson)
}
