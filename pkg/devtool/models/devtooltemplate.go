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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type SDevtoolTemplate struct {
	SVSCronjob
	Playbook *ansible.Playbook `length:"text" nullable:"false" create:"required" get:"user" update:"user"`
	db.SVirtualResourceBase
}

type SDevtoolTemplateManager struct {
	db.SVirtualResourceBaseManager
}

var (
	DevtoolTemplateManager *SDevtoolTemplateManager
)

func init() {
	DevtoolTemplateManager = &SDevtoolTemplateManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDevtoolTemplate{},
			"devtool_templates_tbl",
			"devtool_template",
			"devtool_templates",
		),
	}
	DevtoolTemplateManager.SetVirtualObject(DevtoolTemplateManager)
}

func (obj *SDevtoolTemplate) PerformBind(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * get playbook struct and create obj
	// * get cronjob struct and create obj
	// * create playbook
	// taskman.TaskManager.NewTask(ctx, "KVMGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)

	task, err := taskman.TaskManager.NewTask(ctx, "TemplateBindingServers", obj, userCred, data.(*jsonutils.JSONDict), "", "", nil)
	if err != nil {
		log.Errorf("register task TemplateBindingServers error %s", err)
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (obj *SDevtoolTemplate) PerformUnbind(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * stop and delete playbook
	// * stop and delete cronjob

	task, err := taskman.TaskManager.NewTask(ctx, "TemplateUnbindingServers", obj, userCred, data.(*jsonutils.JSONDict), "", "", nil)
	if err != nil {
		log.Errorf("register task TemplateUnbindingServers error %s", err)
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (obj *SDevtoolTemplate) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	obj.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	task, err := taskman.TaskManager.NewTask(ctx, "TemplateUpdate", obj, userCred, data.(*jsonutils.JSONDict), "", "", nil)
	if err != nil {
		log.Errorf("register task TemplateUpdate error %s", err)
	}
	task.ScheduleRun(nil)
}

func (obj *SDevtoolTemplate) ValidateDeleteCondition(ctx context.Context) error {

	template := obj
	items := make([]SCronjob, 0)
	q := CronjobManager.Query().Equals("template_id", template.Id)
	err := q.All(&items)
	if err != nil {
		log.Errorf("query cronjobs for %s error: %s", template.Id, err)
		return err
	}
	if len(items) > 0 {
		return httperrors.NewNotEmptyError("devtooltemplate")
	}
	return nil
}
