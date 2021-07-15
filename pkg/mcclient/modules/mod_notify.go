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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type ConfigsManager struct {
	modulebase.ResourceManager
}

var (
	NotifyReceiver   modulebase.ResourceManager
	NotifyConfig     modulebase.ResourceManager
	NotifyRobot      modulebase.ResourceManager
	Notification     modulebase.ResourceManager
	NotifyTemplate   modulebase.ResourceManager
	NotifyTopic      modulebase.ResourceManager
	NotifySubscriber modulebase.ResourceManager
	Configs          ConfigsManager
)

func init() {
	NotifyReceiver = NewNotifyv2Manager(
		"receiver",
		"receivers",
		[]string{"ID", "Name", "Domain_Id", "Project_Domain", "Email", "International_Mobile", "Enabled_Contact_Types", "Verified_Infos"},
		[]string{},
	)
	register(&NotifyReceiver)

	NotifyConfig = NewNotifyv2Manager(
		"notifyconfig",
		"notifyconfigs",
		[]string{"Name", "Type", "Content", "Attribution", "Project_Domain"},
		[]string{},
	)
	register(&NotifyConfig)

	NotifyRobot = NewNotifyv2Manager(
		"robot",
		"robots",
		[]string{"ID", "Name", "Type", "Address", "Lang"},
		[]string{},
	)
	register(&NotifyRobot)

	Notification = NewNotifyv2Manager(
		"notification",
		"notifications",
		[]string{"Title", "Content", "ContactType", "Priority", "Receiver_Details"},
		[]string{},
	)
	register(&Notification)

	NotifyTemplate = NewNotifyv2Manager(
		"notifytemplate",
		"notifytemplates",
		[]string{"ID", "Name", "Contact_Type", "Topic", "Template_Type", "Content", "Example"},
		[]string{},
	)
	register(&NotifyTemplate)

	NotifyTopic = NewNotifyv2Manager(
		"topic",
		"topics",
		[]string{"ID", "Name", "Type", "Enabled", "Resources"},
		[]string{},
	)
	register(&NotifyTopic)

	NotifySubscriber = NewNotifyv2Manager(
		"subscriber",
		"subscribers",
		[]string{"ID", "Name", "Topic_Id", "Type", "Resource_Scope", "Role_Scope", "Receivers", "Role", "Robot"},
		[]string{},
	)
	register(&NotifySubscriber)
}
