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

type SubscriberCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	// description: Id of Topic
	// required
	TopicID string `json:"topic_id"`

	// description: scope of resource
	// enum: ["system","domain","project"]
	ResourceScope string `json:"resource_scope"`

	// description: project id or domain id of resource
	// example: 1e3824756bac4ac084e784ed297ec652
	ResourceAttributionId string `json:"resource_attribution_id"`

	ResourceAttributionName string `json:"resource_attribution_name"`

	// description: domain id of resource
	// example: 1e3824756bac4ac084e784ed297ec652
	DomainId string `json:"domain_id"`

	// description: Type of subscriber
	// enum: ["receiver","robot","role"]
	Type string `json:"type"`

	// description: receivers which is required when the type is 'receiver' will Subscribe TopicID
	Receivers []string `json:"receivers"`

	// description: Role(Id or Name) which is required when the type is 'role' will Subscribe TopicID
	Role string `json:"role"`

	// description: The scope of role subscribers
	// enum: ["system","domain","project"]
	RoleScope string `json:"role_scope"`

	// description: Robot(Id or Name) which is required when the type is 'robot' will Subscribe TopicID
	Robot string `json:"robot"`

	// description: scope
	// enum: ["system","domain"]
	Scope string `json:"scope"`
	// minutes
	GroupTimes *uint32 `json:"group_times"`
}

type SubscriberChangeInput struct {
	// description: receivers which is required when the type is 'receiver' will Subscribe TopicID
	Receivers []string `json:"receivers"`

	// description: Role(Id or Name) which is required when the type is 'role' will Subscribe TopicID
	Role string `json:"role"`

	// description: The scope of role subscribers
	// enum: ["system","domain","project"]
	RoleScope string `json:"role_scope"`

	// description: Robot(Id or Name) which is required when the type is 'robot' will Subscribe TopicID
	Robot string `json:"robot"`
	// minutes
	GroupTimes *uint32 `json:"group_times"`
}

type SubscriberListInput struct {
	apis.StandaloneAnonResourceListInput
	apis.EnabledResourceBaseListInput

	// description: topic id
	TopicID string `json:"topic_id"`

	// description: scope of resource
	// enum: ["system","domain","project"]
	ResourceScope string `json:"resource_scope"`

	// description: type
	// enum: ["receiver","robot","role"]
	Type string `json:"type"`

	// description: scope
	// enum: ["system","domain"]
	Scope string `json:"scope"`
}

type Identification struct {
	// example: 036fed49483b412888a760c2bc995caa
	ID string `json:"id"`
	// example: test
	Name string `json:"name"`
}

type SubscriberDetails struct {
	apis.StandaloneAnonResourceDetails
	SSubscriber

	// description: receivers
	Receivers []Identification `json:"receivers"`

	// description: role
	Role Identification `json:"role"`

	// description: robot
	Robot Identification `json:"robot"`
}

type SubscriberSetReceiverInput struct {
	Receivers []string `json:"receivers"`
}
