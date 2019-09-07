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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Notifications NotificationManager
)

type SNotifyMessage struct {
	Uid         string          `json:"uid,omitempty"`
	Gid         string          `json:"gid,omitempty"`
	ContactType TNotifyChannel  `json:"contact_type,omitempty"`
	Topic       string          `json:"topic,omitempty"`
	Priority    TNotifyPriority `json:"priority,omitempty"`
	Msg         string          `json:"msg,omitempty"`
	Remark      string          `json:"remark,omitempty"`
	Broadcast   bool            `json:"broadcast,omitempty"`
}

type NotificationManager struct {
	modulebase.ResourceManager
}

func (manager *NotificationManager) Send(s *mcclient.ClientSession, msg SNotifyMessage) error {
	_, err := manager.Create(s, jsonutils.Marshal(&msg))
	return err
}

func init() {
	Notifications = NotificationManager{
		modules.NewNotifyManager("notification", "notifications",
			[]string{"id", "uid", "contact_type", "topic", "priority", "msg", "received_at", "send_by", "status", "create_at", "update_at", "delete_at", "create_by", "update_by", "delete_by", "is_deleted", "broadcast", "remark"},
			[]string{}),
	}

	modules.Register(&Notifications)
}
