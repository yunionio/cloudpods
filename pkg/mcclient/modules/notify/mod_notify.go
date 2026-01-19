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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	notify_apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ReceiverManager struct {
	modulebase.ResourceManager
}

type ConfigsManager struct {
	modulebase.ResourceManager
}

var (
	NotifyReceiver   ReceiverManager
	NotifyConfig     modulebase.ResourceManager
	NotifyRobot      modulebase.ResourceManager
	Notification     modulebase.ResourceManager
	NotifyTemplate   modulebase.ResourceManager
	NotifyTopic      modulebase.ResourceManager
	NotifySubscriber modulebase.ResourceManager
	Configs          ConfigsManager
)

func (rm *ReceiverManager) SyncUserContact(s *mcclient.ClientSession, userId string, mobile string, email string) error {
	recv, err := rm.GetById(s, userId, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			// not created yet, do create
			newRecv := notify_apis.ReceiverCreateInput{
				UID: userId,
			}
			if len(mobile) > 0 {
				newRecv.InternationalMobile.Mobile = mobile
				newRecv.EnabledContactTypes = append(newRecv.EnabledContactTypes, "mobile")
			}
			if len(email) > 0 {
				newRecv.Email = email
				newRecv.EnabledContactTypes = append(newRecv.EnabledContactTypes, "email")
			}
			_, err := rm.Create(s, jsonutils.Marshal(newRecv))
			if err != nil {
				// create failed
				return errors.Wrap(err, "create receiver")
			}
			// success
		} else {
			return errors.Wrap(err, "GetById")
		}
	} else {
		// receiver exists
		fmt.Println("receiver exists", recv.String())
		recvObj := notify_apis.ReceiverDetails{}
		err := recv.Unmarshal(&recvObj)
		if err != nil {
			return errors.Wrap(err, "Unmarshal ReceiverDetails")
		}
		updateRecv := notify_apis.ReceiverUpdateInput{}
		changeMobile := false
		changeEmail := false
		if mobile != "" && recvObj.InternationalMobile.Mobile != mobile {
			updateRecv.InternationalMobile.Mobile = mobile
			updateRecv.EnabledContactTypes = append(updateRecv.EnabledContactTypes, "mobile")
			changeMobile = true
		}
		if email != "" && recvObj.Email != email {
			updateRecv.Email = email
			updateRecv.EnabledContactTypes = append(updateRecv.EnabledContactTypes, "email")
			changeEmail = true
		}
		if changeMobile || changeEmail {
			_, err = rm.Update(s, userId, jsonutils.Marshal(updateRecv))
			if err != nil {
				return errors.Wrap(err, "update receiver")
			}
		}
	}

	return nil
}

func init() {
	NotifyReceiver = ReceiverManager{
		ResourceManager: modules.NewNotifyv2Manager(
			"receiver",
			"receivers",
			[]string{"ID", "Name", "Domain_Id", "Project_Domain", "Email", "International_Mobile", "Enabled_Contact_Types", "Verified_Infos"},
			[]string{},
		),
	}
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
		[]string{"ID", "Name", "Contact_Type", "Title", "Content", "Priority", "Status", "Received_At", "Receiver_Type"},
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

	// important: Notifications' init must be behind Notication's init
	Notifications = NotificationManager{
		Notification,
	}
}
