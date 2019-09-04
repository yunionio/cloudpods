package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
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
	db.RegisterModelManager(DevtoolTemplateManager)
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
