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
	"yunion.io/x/pkg/errors"

	ansible_api "yunion.io/x/onecloud/pkg/apis/ansibleserver"
	devtool_api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ansible_modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
)

type SwitchEditionTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SwitchEditionTask{})
}

func (self *SwitchEditionTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sw := obj.(*models.SEditionSwitch)
	probe, err := models.ProbeControlPlaneSSH(ctx)
	if err != nil {
		self.fail(ctx, sw, err)
		return
	}
	if !probe.Sshable {
		self.fail(ctx, sw, errors.Error(fmt.Sprintf("control plane is not sshable: %s", probe.Reason)))
		return
	}
	if sw.Host != probe.Host {
		if _, err := db.Update(sw, func() error {
			sw.Host = probe.Host
			return nil
		}); err != nil {
			self.fail(ctx, sw, errors.Wrap(err, "record control plane host"))
			return
		}
	}
	if err := models.RemoveOcbootSwitchLog(ctx, probe.Host); err != nil {
		self.fail(ctx, sw, errors.Wrap(err, "remove previous ocboot switch log"))
		return
	}

	session := auth.GetAdminSession(ctx, "")
	ref, err := ansible_modules.AnsiblePlaybookReference.GetByName(session, devtool_api.OcbootPlaybookReferenceName, nil)
	if err != nil {
		self.fail(ctx, sw, errors.Wrap(err, "get ocboot ansible playbook reference"))
		return
	}
	refId, _ := ref.GetString("id")
	if len(refId) == 0 {
		self.fail(ctx, sw, errors.Error("ocboot ansible playbook reference has no id"))
		return
	}

	args := jsonutils.NewDict()
	argv := jsonutils.NewArray()
	for _, arg := range []string{
		"switch-edition",
		probe.Host,
		sw.Edition,
		"-u", devtool_api.OcbootSSHUser,
		"-p", fmt.Sprintf("%d", devtool_api.OcbootSSHPort),
	} {
		argv.Add(jsonutils.NewString(arg))
	}
	args.Set("ocboot_argv", argv)

	params := jsonutils.NewDict()
	params.Set("args", args)
	params.Set("host", jsonutils.Marshal(ansible_api.AnsibleHost{
		User: devtool_api.OcbootSSHUser,
		IP:   probe.Host,
		Port: devtool_api.OcbootSSHPort,
		Name: "primary-master",
	}))

	self.SetStage("OnAnsiblePlaybookComplete", nil)
	taskHeader := self.GetTaskRequestHeader()
	session.Header.Set(mcclient.TASK_NOTIFY_URL, taskHeader.Get(mcclient.TASK_NOTIFY_URL))
	session.Header.Set(mcclient.TASK_ID, taskHeader.Get(mcclient.TASK_ID))
	ret, err := ansible_modules.AnsiblePlaybookReference.PerformAction(session, refId, "run", params)
	if err != nil {
		self.fail(ctx, sw, errors.Wrap(err, "run ocboot playbook reference"))
		return
	}
	if instanceId, _ := ret.GetString("ansible_playbook_instance_id"); len(instanceId) > 0 {
		sw.SetAnsibleInstance(instanceId)
	}
}

func (self *SwitchEditionTask) fail(ctx context.Context, sw *models.SEditionSwitch, err error) {
	if markErr := sw.MarkFailed(err.Error()); markErr != nil {
		err = errors.Wrapf(err, "mark edition switch failed: %v", markErr)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SwitchEditionTask) OnAnsiblePlaybookComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sw := obj.(*models.SEditionSwitch)
	if err := sw.MarkSucceeded(); err != nil {
		self.fail(ctx, sw, err)
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *SwitchEditionTask) OnAnsiblePlaybookCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sw := obj.(*models.SEditionSwitch)
	reason := "ocboot switch-edition failed"
	if body != nil {
		reason = body.String()
	}
	self.fail(ctx, sw, errors.Error(reason))
}
