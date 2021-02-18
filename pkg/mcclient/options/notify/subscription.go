// Copyright 2019 Yunion
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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SubscriptionListOptions struct {
	options.BaseListOptions
}

func (opts *SubscriptionListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SubscriptionOptions struct {
	ID string `help:"Id or Name of subscription"`
}

func (so *SubscriptionOptions) GetId() string {
	return so.ID
}

func (so *SubscriptionOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SubscriptionSetReceiverOptions struct {
	SubscriptionOptions
	Receivers     []string `json:"receivers"`
	RoleAndScopes []string `json:"role_and_scopes" help:"role and scope, separated by a colon, example: admin:system"`
}

func (opts *SubscriptionSetReceiverOptions) Params() (jsonutils.JSONObject, error) {
	d := jsonutils.NewDict()
	if len(opts.Receivers) > 0 {
		d.Set("receivers", jsonutils.NewStringArray(opts.Receivers))
	}
	if len(opts.RoleAndScopes) > 0 {
		ra := jsonutils.NewArray()
		for _, rs := range opts.RoleAndScopes {
			index := strings.Index(rs, ":")
			if index <= 0 {
				return nil, fmt.Errorf("invalid role_and_scope %q, example: %s", rs, "admin:system")
			}
			rd := jsonutils.NewDict()
			rd.Set("role", jsonutils.NewString(rs[:index]))
			rd.Set("scope", jsonutils.NewString(rs[index+1:]))
			ra.Add(rd)
		}
		d.Set("roles", ra)
	}
	return d, nil
}

type SsubscriptionSetRobotOptions struct {
	ROBOT string `choices:"feishu-robot|dingtalk-robot|workwx-robot"`
}

type SubscriptionSetRobotOptions struct {
	SubscriptionOptions
	SsubscriptionSetRobotOptions
}

func (opts *SubscriptionSetRobotOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SsubscriptionSetRobotOptions), nil
}

type SsubscriptionSetWebhookOptions struct {
	WEBHOOK string `choices:"webhook"`
}

type SubscriptionSetWebhookOptions struct {
	SubscriptionOptions
	SsubscriptionSetWebhookOptions
}

func (opts *SubscriptionSetWebhookOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SsubscriptionSetWebhookOptions), nil
}
