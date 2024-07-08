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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/uefi"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type IServerBaseDeployTask interface {
	IPXEBootTask

	RemoveEFIOSEntry() bool
	DoDeploys(ctx context.Context, term *ssh.Client) (jsonutils.JSONObject, error)
	PostDeploys(ctx context.Context, term *ssh.Client) error
}

type SBaremetalServerBaseDeployTask struct {
	SBaremetalPXEBootTaskBase

	needPXEBoot bool
}

func newBaremetalServerBaseDeployTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) SBaremetalServerBaseDeployTask {
	task := SBaremetalServerBaseDeployTask{
		SBaremetalPXEBootTaskBase: newBaremetalPXEBootTaskBase(userCred, baremetal, taskId, data),
		needPXEBoot:               true,
	}
	// any inheritance must call:
	// task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalServerBaseDeployTask) IServerBaseDeployTask() IServerBaseDeployTask {
	return self.GetVirtualObject().(IServerBaseDeployTask)
}

func (self *SBaremetalServerBaseDeployTask) GetName() string {
	return "BaremetalServerBaseDeployTask"
}

func (self *SBaremetalServerBaseDeployTask) NeedPXEBoot() bool {
	return self.needPXEBoot
}

func (self *SBaremetalServerBaseDeployTask) IsDisableImageCache() bool {
	return jsonutils.QueryBoolean(self.data, "disable_image_cache", false)
}

func (self *SBaremetalServerBaseDeployTask) GetFinishAction() string {
	if self.data != nil {
		action, _ := self.data.GetString("on_finish")
		return action
	}
	return ""
}

func (self *SBaremetalServerBaseDeployTask) RemoveEFIOSEntry() bool {
	return false
}

func (self *SBaremetalServerBaseDeployTask) DoDeploys(ctx context.Context, _ *ssh.Client) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (self *SBaremetalServerBaseDeployTask) PostDeploys(_ context.Context, _ *ssh.Client) error {
	return nil
}

func (self *SBaremetalServerBaseDeployTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	log.Infof("%s called on stage pxeboot, args: %v", self.GetName(), args)

	if self.IServerBaseDeployTask().RemoveEFIOSEntry() {
		if err := uefi.RemoteTryRemoveOSBootEntry(term); err != nil {
			return errors.Wrap(err, "Remote uefi boot entry")
		}
	}

	result, err := self.IServerBaseDeployTask().DoDeploys(ctx, term)
	if err != nil {
		return errors.Wrap(err, "Do deploy")
	}

	if err := AdjustUEFIBootOrder(ctx, term, self.Baremetal); err != nil {
		return errors.Wrap(err, "Adjust UEFI boot order")
	}

	_, err = term.Run(
		"/bin/sync",
		"/sbin/sysctl -w vm.drop_caches=3",
	)
	if err != nil {
		return errors.Wrap(err, "Sync disk")
	}

	if err := self.IServerBaseDeployTask().PostDeploys(ctx, term); err != nil {
		return errors.Wrap(err, "post deploy")
	}

	onFinishAction := self.GetFinishAction()
	if utils.IsInStringArray(onFinishAction, []string{"restart", "shutdown"}) {
		if self.Baremetal.HasBMC() {
			if err := self.EnsurePowerShutdown(false); err != nil {
				return errors.Wrap(err, "Ensure power off")
			}
			if onFinishAction == "restart" {
				if err := self.EnsurePowerUp(); err != nil {
					return errors.Wrap(err, "Ensure power up")
				}
			}
			self.Baremetal.AutoSyncAllStatus(ctx)
		} else {
			if onFinishAction == "shutdown" {
				log.Infof("None BMC baremetal can't shutdown when deploying")
				/*
				 * if err := self.Baremetal.SSHShutdown(); err != nil {
				 *     return errors.Wrap(err, "Try ssh shutdown")
				 * }
				 */
			} else {
				// do restart
				// hack: ssh reboot to disk
				self.needPXEBoot = false
				if err := self.EnsureSSHReboot(ctx); err != nil {
					return errors.Wrap(err, "Try ssh reboot")
				}
			}
		}
		self.Baremetal.SyncAllStatus(ctx, types.POWER_STATUS_ON)
	}
	SetTaskComplete(self, result)
	return nil
}
