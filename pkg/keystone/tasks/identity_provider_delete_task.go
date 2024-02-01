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

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type IdentityProviderDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(IdentityProviderDeleteTask{})
}

func (self *IdentityProviderDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	idp := obj.(*models.SIdentityProvider)

	err := idp.Purge(ctx, self.UserCred)
	if err != nil {
		idp.SetStatus(ctx, self.UserCred, api.IdentityDriverStatusDeleteFailed, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("purge failed %s", err)))
		logclient.AddActionLogWithStartable(self, idp, logclient.ACT_DELETE, err, self.UserCred, false)
		return
	}

	logclient.AddActionLogWithStartable(self, idp, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
