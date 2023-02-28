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

package sender

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/ansibleserver/options"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/websocket"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWebsocketSender struct {
	config map[string]api.SNotifyConfigContent
}

func (websocket *SWebsocketSender) GetSenderType() string {
	return api.WEBSOCKET
}

func (websocket *SWebsocketSender) Send(params api.SendParams) error {
	return websocket.send(params)
}

func (websocket *SWebsocketSender) send(args api.SendParams) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("notify"), "obj_type")
	params.Add(jsonutils.NewString(""), "obj_id")
	params.Add(jsonutils.NewString(""), "obj_name")
	params.Add(jsonutils.JSONTrue, "success")
	// component request body
	body := jsonutils.DeepCopy(params).(*jsonutils.JSONDict)
	body.Add(jsonutils.NewString(args.Title), "action")
	body.Add(jsonutils.NewString(fmt.Sprintf("priority=%s; content=%s", args.Priority, args.Message)), "notes")
	body.Add(jsonutils.NewString(args.Receivers.Contact), "user_id")
	body.Add(jsonutils.NewString(args.Receivers.Contact), "user")
	if len(args.Receivers.Contact) == 0 {
		body.Add(jsonutils.JSONTrue, "broadcast")
	}
	if websocket.isFailed(args.Title, args.Message) {
		body.Add(jsonutils.JSONFalse, "success")
	}
	session := auth.GetAdminSession(context.Background(), options.Options.Region)
	_, err := modules.Websockets.Create(session, body)
	if err != nil {
		// failed
		_, err = modules.Websockets.Create(session, body)

		return err
	}
	return nil
}

func (websocket *SWebsocketSender) isFailed(title, message string) bool {
	for _, c := range []string{title, message} {
		for _, k := range FAIL_KEY {
			if strings.Contains(c, k) {
				return true
			}
		}
	}
	return false
}

func (websocket *SWebsocketSender) IsPersonal() bool {
	return true
}

func (websocket *SWebsocketSender) IsRobot() bool {
	return false
}

func (websocket *SWebsocketSender) IsValid() bool {
	return true
}

func (websocket *SWebsocketSender) IsPullType() bool {
	return true
}

func (websocket *SWebsocketSender) IsSystemConfigContactType() bool {
	return true
}

func (websocket *SWebsocketSender) ContactByMobile(mobile, domainId string) (string, error) {
	return "", nil
}

func (websocket *SWebsocketSender) GetAccessToken(key string) error {
	return nil
}

func (websocket *SWebsocketSender) ValidateConfig(config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func init() {
	models.Register(&SWebsocketSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
