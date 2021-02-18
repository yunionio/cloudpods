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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type NotificationCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// description: ids or names of receiver
	// required: false
	// example: {"adfb720ccdd34c638346ea4fa7a713a8", "zhangsan"}
	Receivers []string `json:"receivers"`
	// description: direct contact, admin privileges required
	// required: false
	Contacts []string `json:"contacts"`
	// description: contact type
	// required: ture
	// example: email
	ContactType string `json:"contact_type"`
	// description: notification topic
	// required: true
	// example: IMAGE_ACTIVE
	Topic string `json:"topic"`
	// description: notification priority
	// required: false
	// enum: fatal,important,nomal
	// example: normal
	Priority string `json:"priority"`
	// description: message content or jsonobject
	// required: ture
	Message string `json:"message"`
	// description: notification tag
	// required: false
	// example: alert
	Tag                       string                 `json:"tag"`
	Metadata                  map[string]interface{} `json:"metadata"`
	IgnoreNonexistentReceiver bool                   `json:"ignore_nonexistent_receiver"`
}

type ReceiveDetail struct {
	ReceiverId   string    `json:"receiver_id"`
	ReceiverName string    `json:"receiver_name"`
	Contact      string    `json:"contact"`
	SendAt       time.Time `json:"sendAt"`
	SendBy       string    `json:"send_by"`
	Status       string    `json:"status"`
	FailedReason string    `json:"failed_reason"`
}

type NotificationDetails struct {
	apis.StatusStandaloneResourceDetails

	SNotification

	Title          string          `json:"title"`
	Content        string          `json:"content"`
	ReceiveDetails []ReceiveDetail `json:"receive_details"`
}

type NotificationListInput struct {
	apis.StatusStandaloneResourceListInput

	ContactType string
	ReceiverId  string
	Tag         string
}

type NotificationManagerEventNotifyInput struct {
	// description: ids or names of receiver
	// required: false
	// example: {"adfb720ccdd34c638346ea4fa7a713a8", "zhangsan"}
	Receivers []string `json:"receivers"`
	// description: direct contact, admin privileges required
	// required: false
	Contacts []string `json:"contacts"`
	// description: contact types
	// required: false
	// example: email
	ContactTypes []string `json:"contact_type"`
	// description: notification priority
	// required: false
	// enum: fatal,important,nomal
	// example: normal
	Priority string `json:"priority"`
	// description: resource details
	// required: ture
	ResourceDetails *jsonutils.JSONDict `json:"resource_details"`
	// description: event trigger sending notification
	// required: true
	// example: SERVER_DELETE
	Event string `json:"event"`
	// description: day left before the event
	// required: false
	// example: 0
	AdvanceDays int `json:"advance_days"`
	// description: domainId of the resource that triggered the event notification
	// required: false
	// example: default
	ProjectDomainId string
	// description: projectId of the resource that triggered the event notification
	// required: false
	// example: f627e09f038645f08ce6880c8d9cb8fd
	ProjectId string `json:"project_id"`
}

type NotificationManagerEventNotifyOutput struct {
	FailedList []FailedElem
}

type FailedElem struct {
	ContactType string
	Reason      string
}
