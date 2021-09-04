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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/devtool/utils"
)

type ServiceUrlCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ServiceUrlCreateTask{})
}

func (self *ServiceUrlCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	serviceUrl := obj.(*models.SServiceUrl)
	info, err := utils.GetServerInfo(ctx, serviceUrl.ServerId)
	if err != nil {
		serviceUrl.MarkCreateFailed(fmt.Sprintf("unable to get serverInfo of server %s: %v", serviceUrl.ServerId, err))
		self.SetStageFailed(ctx, nil)
		return
	}

	url, err := utils.GetServiceUrl(ctx, serviceUrl.Service)
	if err != nil {
		serviceUrl.MarkCreateFailed(err.Error())
		self.SetStageFailed(ctx, nil)
		return
	}
	url, err = utils.FindValidServiceUrl(ctx, utils.Service{
		Url:  url,
		Name: serviceUrl.Service,
	}, "", info, &ansible.AnsibleHost{
		User: serviceUrl.ServerAnsibleInfo.User,
		IP:   serviceUrl.ServerAnsibleInfo.IP,
		Port: serviceUrl.ServerAnsibleInfo.Port,
		Name: serviceUrl.ServerAnsibleInfo.Name,
	})
	if err != nil {
		serviceUrl.MarkCreateFailed(err.Error())
		self.SetStageFailed(ctx, nil)
		return
	}
	_, err = db.Update(serviceUrl, func() error {
		serviceUrl.Url = url
		serviceUrl.Status = devtool.SERVICEURL_STATUS_READY
		return nil
	})
	if err != nil {
		log.Errorf("unable to update serviceurl: %v", err)
	}
	self.SetStageComplete(ctx, nil)
}
