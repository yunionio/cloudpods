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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	devtool_api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/devtool/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ansible_modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type ApplyScriptTask struct {
	taskman.STask
	cleanFunc func()
}

func init() {
	taskman.RegisterTask(ApplyScriptTask{})
}

func (self *ApplyScriptTask) registerClean(clean func()) {
	self.cleanFunc = clean
}

func (self *ApplyScriptTask) clean() {
	if self.cleanFunc == nil {
		return
	}
	self.cleanFunc()
}

func (self *ApplyScriptTask) taskFailed(ctx context.Context, sa *models.SScriptApply, sar *models.SScriptApplyRecord, err error) {
	self.clean()
	var failCode string
	switch errors.Cause(err) {
	case utils.ErrServerNotSshable:
		failCode = devtool_api.SCRIPT_APPLY_RECORD_FAILCODE_SSHABLE
	case utils.ErrCannotReachInfluxbd:
		failCode = devtool_api.SCRIPT_APPLY_RECORD_FAILCODE_INFLUXDB
	default:
		failCode = devtool_api.SCRIPT_APPLY_RECORD_FAILCODE_OTHERS
	}
	err = sa.StopApply(self.UserCred, sar, false, failCode, err.Error())
	if err != nil {
		log.Errorf("unable to StopApply script %s to server %s", sa.ScriptId, sa.GuestId)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	if failCode == devtool_api.SCRIPT_APPLY_RECORD_FAILCODE_OTHERS {
		// restart
		err = sa.StartApply(ctx, self.UserCred)
		if err != nil {
			log.Errorf("unable to StartApply script %s to server %s", sa.ScriptId, sa.GuestId)
		}
	}
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	self.SetStageFailed(ctx, jsonutils.NewString(errMsg))
}

func (self *ApplyScriptTask) taskSuccess(ctx context.Context, sa *models.SScriptApply, sar *models.SScriptApplyRecord) {
	self.clean()
	err := sa.StopApply(self.UserCred, sar, true, "", "")
	if err != nil {
		log.Errorf("unable to StopApply script %s to server %s", sa.ScriptId, sa.GuestId)
		self.SetStageComplete(ctx, nil)
	}
}

func (self *ApplyScriptTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sa := obj.(*models.SScriptApply)
	// create record
	sar, err := models.ScriptApplyRecordManager.CreateRecord(ctx, sa.GetId())
	if err != nil {
		self.taskFailed(ctx, sa, nil, err)
		return
	}
	s, err := sa.Script()
	if err != nil {
		self.taskFailed(ctx, sa, sar, err)
		return
	}
	session := auth.GetAdminSession(ctx, "")
	sshable, cleanFunc, err := utils.CheckSSHable(session, sa.GuestId)
	// check sshable
	if err != nil {
		self.taskFailed(ctx, sa, sar, err)
		return
	}
	if cleanFunc != nil {
		self.registerClean(func() {
			err := cleanFunc()
			if err != nil {
				log.Errorf("unable to clean: %v", err)
			}
		})
	}

	host := ansible_api.AnsibleHost{
		User:     sshable.User,
		IP:       sshable.Host,
		Port:     sshable.Port,
		Name:     sshable.ServerName,
		Password: sshable.Password,
		OsType:   sshable.OsType,
	}

	// genrate args
	params := jsonutils.NewDict()
	if len(sa.ArgsGenerator) == 0 {
		params.Set("args", sa.Args)
	} else {
		generator, ok := utils.GetArgGenerator(sa.ArgsGenerator)
		if !ok {
			params.Set("args", sa.Args)
		}
		arg, err := generator(ctx, sa.GuestId, sshable.ProxyEndpointId, &host)
		if err != nil {
			self.taskFailed(ctx, sa, sar, err)
			return
		}
		params.Set("args", jsonutils.Marshal(arg))
	}

	params.Set("host", jsonutils.Marshal(host))

	// fetch ansible playbook reference id
	updateData := jsonutils.NewDict()
	updateData.Set("script_apply_record_id", jsonutils.NewString(sar.GetId()))
	self.SetStage("OnAnsiblePlaybookComplete", updateData)

	// Inject Task Header
	taskHeader := self.GetTaskRequestHeader()
	session.Header.Set(mcclient.TASK_NOTIFY_URL, taskHeader.Get(mcclient.TASK_NOTIFY_URL))
	session.Header.Set(mcclient.TASK_ID, taskHeader.Get(mcclient.TASK_ID))
	_, err = ansible_modules.AnsiblePlaybookReference.PerformAction(session, s.PlaybookReferenceId, "run", params)
	if err != nil {
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "can't run ansible playbook reference %s", s.PlaybookReferenceId))
		return
	}
}

func mapStringSlice(f func(string) string, a []string) []string {
	for i := range a {
		a[i] = f(a[i])
	}
	return a
}

const (
	agentInstalledKey   = "__monitor_agent"
	agentInstalledValue = "true"
)

func (self *ApplyScriptTask) OnAnsiblePlaybookComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	// try to delete local forward
	session := auth.GetAdminSession(ctx, "")

	sa := obj.(*models.SScriptApply)
	// try to set metadata for guest
	metadata := jsonutils.NewDict()
	metadata.Set(agentInstalledKey, jsonutils.NewString(agentInstalledValue))
	_, err := compute.Servers.PerformAction(session, sa.GuestId, "metadata", metadata)
	if err != nil {
		log.Errorf("set metadata '%s:%s' for guest %s failed: %v", agentInstalledKey, agentInstalledValue, sa.GuestId, err)
	}

	sarId, _ := self.Params.GetString("script_apply_record_id")
	osar, err := models.ScriptApplyRecordManager.FetchById(sarId)
	if err != nil {
		log.Errorf("unable to fetch script apply record %s: %v", sarId, err)
		self.taskSuccess(ctx, sa, nil)
	}
	self.taskSuccess(ctx, sa, osar.(*models.SScriptApplyRecord))
}

func (self *ApplyScriptTask) OnAnsiblePlaybookCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sa := obj.(*models.SScriptApply)
	sarId, _ := self.Params.GetString("script_apply_record_id")
	osar, err := models.ScriptApplyRecordManager.FetchById(sarId)
	if err != nil {
		log.Errorf("unable to fetch script apply record %s: %v", sarId, err)
		self.taskFailed(ctx, sa, nil, errors.Error(body.String()))
	} else {
		self.taskFailed(ctx, sa, osar.(*models.SScriptApplyRecord), errors.Error(body.String()))
	}
}
