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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

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
	NotifyReceiver = modules.NewNotifyv2Manager(
		"receiver",
		"receivers",
		[]string{"ID", "Name", "Domain_Id", "Project_Domain", "Email", "International_Mobile", "Enabled_Contact_Types", "Verified_Infos"},
		[]string{},
	)
	modules.Register(&NotifyReceiver)

	NotifyConfig = modules.NewNotifyv2Manager(
		"notifyconfig",
		"notifyconfigs",
		[]string{"Name", "Type", "Content", "Attribution", "Project_Domain"},
		[]string{},
	)
	modules.Register(&NotifyConfig)

	NotifyRobot = modules.NewNotifyv2Manager(
		"robot",
		"robots",
		[]string{"ID", "Name", "Type", "Address", "Lang"},
		[]string{},
	)
	modules.Register(&NotifyRobot)

	Notification = modules.NewNotifyv2Manager(
		"notification",
		"notifications",
		[]string{"Title", "Content", "ContactType", "Priority", "Receiver_Details"},
		[]string{},
	)
	modules.Register(&Notification)

	NotifyTemplate = modules.NewNotifyv2Manager(
		"notifytemplate",
		"notifytemplates",
		[]string{"ID", "Name", "Contact_Type", "Topic", "Template_Type", "Content", "Example"},
		[]string{},
	)
	modules.Register(&NotifyTemplate)

	NotifyTopic = modules.NewNotifyv2Manager(
		"topic",
		"topics",
		[]string{"ID", "Name", "Type", "Enabled", "Resources"},
		[]string{},
	)
	modules.Register(&NotifyTopic)

	NotifySubscriber = modules.NewNotifyv2Manager(
		"subscriber",
		"subscribers",
		[]string{"ID", "Name", "Topic_Id", "Type", "Resource_Scope", "Role_Scope", "Receivers", "Role", "Robot"},
		[]string{},
	)
	modules.Register(&NotifySubscriber)
}
