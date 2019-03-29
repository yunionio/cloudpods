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
