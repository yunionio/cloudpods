// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed unde3r the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
)

type TemplateBindingServers struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(TemplateBindingServers{})
}

func (self *TemplateBindingServers) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {

	template := obj.(*models.SDevtoolTemplate)
	_, err := template.Binding(ctx, self.UserCred, nil, self.Params)
	if err != nil {
		self.SetStageFailed(ctx, fmt.Sprintf("TemplateUpdate failed %s", err))
	} else {
		self.SetStageComplete(ctx, nil)
	}
}
