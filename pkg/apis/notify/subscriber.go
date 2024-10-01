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
	TopicID string

	// description: scope of resource
	// enum: ["system","domain","project"]
	ResourceScope string

	// description: project id or domain id of resource
	// example: 1e3824756bac4ac084e784ed297ec652
	ResourceAttributionId string

	ResourceAttributionName string

	// description: domain id of resource
	// example: 1e3824756bac4ac084e784ed297ec652
	DomainId string

	// description: Type of subscriber
	// enum: ["receiver","robot","role"]
	Type string

	// description: receivers which is required when the type is 'receiver' will Subscribe TopicID
	Receivers []string

	// description: Role(Id or Name) which is required when the type is 'role' will Subscribe TopicID
	Role string

	// description: The scope of role subscribers
	// enum: ["system","domain","project"]
	RoleScope string

	// description: Robot(Id or Name) which is required when the type is 'robot' will Subscribe TopicID
	Robot string

	// description: scope
	// enum: ["system","domain"]
	Scope string
	// minutes
	GroupTimes *uint32
}

type SubscriberChangeInput struct {
	// description: receivers which is required when the type is 'receiver' will Subscribe TopicID
	Receivers []string

	// description: Role(Id or Name) which is required when the type is 'role' will Subscribe TopicID
	Role string

	// description: The scope of role subscribers
	// enum: ["system","domain","project"]
	RoleScope string

	// description: Robot(Id or Name) which is required when the type is 'robot' will Subscribe TopicID
	Robot string
	// minutes
	GroupTimes *uint32
}

type SubscriberListInput struct {
	apis.StandaloneAnonResourceListInput
	apis.EnabledResourceBaseListInput

	// description: topic id
	TopicID string

	// description: scope of resource
	// enum: ["system","domain","project"]
	ResourceScope string

	// description: type
	// enum: ["receiver","robot","role"]
	Type string

	// description: scope
	// enum: ["system","domain"]
	Scope string
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
	Receivers []Identification

	// description: role
	Role Identification

	// description: robot
	Robot Identification
}

type SubscriberSetReceiverInput struct {
	Receivers []string
}
