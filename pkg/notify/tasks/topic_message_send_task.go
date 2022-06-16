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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type TopicMessageSendTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(TopicMessageSendTask{})
}

func (self *TopicMessageSendTask) taskFailed(ctx context.Context, topic *models.STopic, err error) {
	logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *TopicMessageSendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	topic := obj.(*models.STopic)
	input := api.NotificationManagerEventNotifyInput{}
	self.GetParams().Unmarshal(&input)
	event, err := topic.CreateEvent(ctx, string(input.ResourceType), string(input.Action), jsonutils.Marshal(input.ResourceDetails).String())
	if err != nil {
		self.taskFailed(ctx, topic, errors.Wrapf(err, "CreateEvent"))
		return
	}
	scribers, err := topic.GetEnabledSubscribers(input.ProjectDomainId, input.ProjectId)
	if err != nil {
		self.taskFailed(ctx, topic, errors.Wrapf(err, "GetSubscribers"))
		return
	}
	receivers := map[string]models.SReceiver{}
	robots := map[string]*models.SRobot{}
	userIds := []string{}
	for i := range scribers {
		switch scribers[i].Type {
		case api.SUBSCRIBER_TYPE_RECEIVER:
			recvs, _ := scribers[i].GetEnabledReceivers()
			for i := range recvs {
				receivers[recvs[i].Id] = recvs[i]
			}
		case api.SUBSCRIBER_TYPE_ROLE:
			query := jsonutils.NewDict()
			query.Set("roles", jsonutils.NewStringArray([]string{scribers[i].Identification}))
			query.Set("effective", jsonutils.JSONTrue)
			if scribers[i].RoleScope == api.SUBSCRIBER_SCOPE_DOMAIN {
				query.Set("project_domain_id", jsonutils.NewString(scribers[i].ResourceAttributionId))
			} else if scribers[i].RoleScope == api.SUBSCRIBER_SCOPE_PROJECT {
				query.Add(jsonutils.NewString(scribers[i].ResourceAttributionId), "scope", "project", "id")
			}
			s := auth.GetAdminSession(ctx, options.Options.Region, "")
			ret, err := identity.RoleAssignments.List(s, query)
			if err != nil {
				logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "RoleAssignments.List"), self.UserCred, false)
				continue
			}
			users := []struct {
				User struct {
					Id string
				}
			}{}
			jsonutils.Update(&users, ret.Data)
			for _, user := range users {
				userIds = append(userIds, user.User.Id)
			}
		case api.SUBSCRIBER_TYPE_ROBOT:
			robot, err := scribers[i].GetRobot()
			if err != nil {
				logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, errors.Wrapf(err, "GetRobot"), self.UserCred, false)
				continue
			}
			if !robot.Enabled.Bool() {
				continue
			}
			robots[robot.Id] = robot
		}
	}

	logclient.AddActionLogWithContext(ctx, topic, logclient.ACT_SEND_NOTIFICATION, jsonutils.Marshal(input), self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
