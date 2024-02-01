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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ansible_modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	"yunion.io/x/onecloud/pkg/util/ansiblev2"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestRestartNetworkTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestRestartNetworkTask{})
}

func (self *GuestRestartNetworkTask) taskFailed(ctx context.Context, guest *models.SGuest, clean func() error, err error) {
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_RESTART_NETWORK_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, jsonutils.NewString(err.Error()), self.UserCred, false)
	if clean != nil {
		err := clean()
		if err != nil {
			log.Errorf("unable to clean: %s", err.Error())
		}
	}
	self.SetStageFailed(ctx, nil)
}

func (self *GuestRestartNetworkTask) OnCloseIpMacSrcCheckComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	ip, _ := self.Params.GetString("ip")
	session := auth.GetAdminSession(ctx, "")
	sshable, clean, err := self.checkSshable(ctx, guest, ip)
	log.Infof("start to CheckSshableForYunionCloud")
	if err != nil {
		self.taskFailed(ctx, guest, clean, err)
		return
	}
	log.Infof("ssable: %s", jsonutils.Marshal(sshable))
	if !sshable.Ok {
		self.taskFailed(ctx, guest, clean, fmt.Errorf("guest %s is not sshable", guest.GetId()))
		return
	}

	playbook := `- hosts: all
  become: true

  tasks:
    - name: "restart network"
      service:
        name: network
        state: restarted
      async: 20
      poll: 0`
	params := jsonutils.NewDict()
	params.Set("playbook", jsonutils.NewString(playbook))

	vars := map[string]interface{}{
		"ansible_port": fmt.Sprintf("%d", sshable.Port),
		"ansible_user": sshable.User,
	}
	host := ansiblev2.NewHost()
	host.Vars = vars
	inv := ansiblev2.NewInventory()
	inv.SetHost(sshable.Host, host)

	params.Set("inventory", jsonutils.NewString(inv.String()))
	params.Set("generate_name", jsonutils.NewString(fmt.Sprintf("%s-restart-network", guest.Name)))

	apb, err := ansible_modules.AnsiblePlaybooksV2.Create(session, params)
	if err != nil {
		self.taskFailed(ctx, guest, clean, err)
		return
	}
	id, _ := apb.GetString("id")
	defer func() {
		_, err := ansible_modules.AnsiblePlaybooksV2.Delete(session, id, nil)
		if err != nil {
			log.Errorf("unable to delete ansibleplaybook %s: %v", id, err)
		}
	}()
	times, waitTimes := 0, time.Second
Loop:
	for times < 10 {
		time.Sleep(waitTimes)
		times++
		waitTimes += time.Second * time.Duration(times)
		apd, err := ansible_modules.AnsiblePlaybooksV2.GetSpecific(session, id, "status", nil)
		if err != nil {
			continue
		}
		status, _ := apd.GetString("status")
		switch status {
		case ansible_api.AnsiblePlaybookStatusInit, ansible_api.AnsiblePlaybookStatusRunning:
			continue
		case ansible_api.AnsiblePlaybookStatusFailed, ansible_api.AnsiblePlaybookStatusCanceled, ansible_api.AnsiblePlaybookStatusUnknown:
			apd, err := ansible_modules.AnsiblePlaybooksV2.GetSpecific(session, id, "output", nil)
			if err != nil {
				self.taskFailed(ctx, guest, nil, errors.Wrapf(err, "ansibleplaybook %s exec failed and can't get its output", id))
				return
			}
			output, _ := apd.GetString("output")
			self.taskFailed(ctx, guest, clean, fmt.Errorf("exec ansibleplaybook failed, its output:\n %s", output))
			return
		case ansible_api.AnsiblePlaybookStatusSucceeded:
			break Loop
		}
	}

	if inBlockStream := jsonutils.QueryBoolean(self.Params, "in_block_stream", false); inBlockStream {
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_BLOCK_STREAM, "")
	} else {
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_RUNNING, "")
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, "", self.UserCred, true)
	if clean != nil {
		err := clean()
		if err != nil {
			log.Errorf("unable to clean: %s", err.Error())
		}
	}

	if !self.Params.Contains("src_ip_check") {
		self.SetStageComplete(ctx, nil)
		return
	}

	srcIpCheck, _ := self.Params.Bool("src_ip_check")
	srcMacCheck, _ := self.Params.Bool("src_mac_check")
	_, err = db.Update(guest, func() error {
		guest.SrcIpCheck = tristate.NewFromBool(srcIpCheck)
		guest.SrcMacCheck = tristate.NewFromBool(srcMacCheck)
		return nil
	})
	if err != nil {
		self.taskFailed(ctx, guest, nil, err)
		return
	}
	self.SetStage("OnResumeIpMacSrcCheckComplete", nil)
	err = guest.StartSyncTask(ctx, self.GetUserCred(), false, self.Id)
	if err != nil {
		self.taskFailed(ctx, guest, nil, err)
	}
}

func (self *GuestRestartNetworkTask) OnResumeIpMacSrcCheckComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestRestartNetworkTask) OnResumeIpMacSrcCheckCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *GuestRestartNetworkTask) OnCloseIpMacSrcCheckCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_RESTART_NETWORK_FAILED, data.String())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, data, self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}

func (self *GuestRestartNetworkTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_RESTART_NETWORK, "restart network")
	if guest.SrcIpCheck.IsTrue() || guest.SrcMacCheck.IsTrue() {
		data := jsonutils.NewDict()
		data.Set("src_ip_check", jsonutils.NewBool(guest.SrcIpCheck.Bool()))
		data.Set("src_mac_check", jsonutils.NewBool(guest.SrcMacCheck.Bool()))
		_, err := db.Update(guest, func() error {
			guest.SrcIpCheck = tristate.False
			guest.SrcMacCheck = tristate.False
			return nil
		})
		if err != nil {
			self.taskFailed(ctx, guest, nil, err)
			return
		}
		self.SetStage("OnCloseIpMacSrcCheckComplete", data)
		err = guest.StartSyncTask(ctx, self.GetUserCred(), false, self.Id)
		if err != nil {
			self.taskFailed(ctx, guest, nil, err)
			return
		}
	} else {
		self.OnCloseIpMacSrcCheckComplete(ctx, obj, data)
	}
}

type SSHable struct {
	Ok     bool
	Reason string

	User string
	Host string
	Port int
}

func (self *GuestRestartNetworkTask) checkSshable(ctx context.Context, guest *models.SGuest, ip string) (sshable SSHable, cleanFunc func() error, err error) {
	vpc, err := guest.GetVpc()
	if err != nil {
		self.taskFailed(ctx, guest, nil, err)
		return
	}
	vpcId := vpc.GetId()
	if vpcId == "" || vpcId == api.DEFAULT_VPC_ID {
		sshable = SSHable{
			Ok:   true,
			User: "cloudroot",
			Host: ip,
			Port: 22,
		}
		return
	}
	lfParams := jsonutils.NewDict()
	lfParams.Set("proto", jsonutils.NewString("tcp"))
	lfParams.Set("port", jsonutils.NewInt(22))
	lfParams.Set("addr", jsonutils.NewString(ip))

	var forward jsonutils.JSONObject
	forward, err = guest.PerformOpenForward(ctx, self.UserCred, nil, lfParams)
	if err != nil {
		err = errors.Wrapf(err, "unable to Open Forward for server %s", guest.Id)
		return
	}
	cleanFunc = func() error {
		proxyAddr := sshable.Host
		proxyPort := sshable.Port
		params := jsonutils.NewDict()
		params.Set("proto", jsonutils.NewString("tcp"))
		params.Set("proxy_addr", jsonutils.NewString(proxyAddr))
		params.Set("proxy_port", jsonutils.NewInt(int64(proxyPort)))
		_, err := guest.PerformCloseForward(ctx, self.UserCred, nil, params)
		if err != nil {
			return errors.Wrapf(err, "unable to close forward(addr %q, port %d, proto %q) for server %s", proxyAddr, proxyPort, "tcp", guest.Id)
		}
		return nil
	}

	proxyAddr, _ := forward.GetString("proxy_addr")
	proxyPort, _ := forward.Int("proxy_port")
	// register
	sshable = SSHable{
		Ok:   true,
		User: "cloudroot",
		Host: proxyAddr,
		Port: int(proxyPort),
	}
	return
}
