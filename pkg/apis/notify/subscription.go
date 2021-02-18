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

import "yunion.io/x/onecloud/pkg/apis"

type SubscriptionListInput struct {
}

type SubscriptionDetails struct {
	apis.StatusStandaloneResourceDetails
	// description: resources managed
	// example: ["server", "eip", "disk"]
	Resources []string `json:"resource_types"`
	// description: receivers of the message sent
	Receivers SubscriptionReceiver `json:"receivers"`
	// description: type of robot send message
	// example: dingtalk_robot
	Robot string `json:"robot"`
	// description: webhook send message
	// example: webhook
	Webhook string `json:"webhook"`
}

type IDAndName struct {
	// example: 036fed49483b412888a760c2bc995caa
	ID string `json:"id"`
	// example: test
	Name string `json:"name"`
}

type ReceivingRoleIDAndName struct {
	IDAndName
	// description: scope of role
	// enum: system,domain,project
	Scope string `json:"scope"`
}

type SubscriptionReceiver struct {
	Receivers      []IDAndName              `json:"receivers"`
	ReceivingRoles []ReceivingRoleIDAndName `json:"roles"`
}

type SubscriptionSetReceiverInput struct {
	ReceivingRoles []ReceivingRole `json:"roles"`
	Receivers      []string        `json:"receivers"`
}

type ReceivingRole struct {
	// description: id or name of role
	Role string `json:"role"`
	// description: scope of role
	// enum: system,domain,project
	Scope string `json:"scope"`
}

type SubscriptionSetRobotInput struct {
	// description: robot
	// enum: feishu-robot,dingtalk-robot,workwx-robot
	Robot string
}

type SubscriptionSetWebhookInput struct {
	Webhook string
}
