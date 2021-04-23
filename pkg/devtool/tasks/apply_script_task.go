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
	"net"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
)

type ApplyScriptTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ApplyScriptTask{})
}

func (self *ApplyScriptTask) taskFailed(ctx context.Context, sa *models.SScriptApply, sar *models.SScriptApplyRecord, err error) {
	err = sa.StopApply(self.UserCred, sar, false, err.Error())
	if err != nil {
		log.Errorf("unable to StopApply script %s to server %s", sa.ScriptId, sa.GuestId)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	// restart
	err = sa.StartApply(ctx, self.UserCred)
	if err != nil {
		log.Errorf("unable to StartApply script %s to server %s", sa.ScriptId, sa.GuestId)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ApplyScriptTask) taskSuccess(ctx context.Context, sa *models.SScriptApply, sar *models.SScriptApplyRecord) {
	err := sa.StopApply(self.UserCred, sar, true, "")
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
	session := auth.GetAdminSession(ctx, "", "")
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	data, err := modules.Servers.GetById(session, sa.GuestId, params)
	if err != nil {
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "unable to fetch server %s", sa.GuestId))
		return
	}
	var serverDetail comapi.ServerDetails
	err = data.Unmarshal(&serverDetail)
	if err != nil {
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "unable to unmarshal %q to ServerDetails", data))
		return
	}

	// check sshable
	sshable, err := self.checkSshable(session, serverDetail.Id)
	if err != nil {
		self.taskFailed(ctx, sa, sar, err)
		return
	}
	if !sshable.ok {
		self.taskFailed(ctx, sa, sar, fmt.Errorf("server %s is not sshable: %s", serverDetail.Id, sshable.reason))
		return
	}
	// make sure user
	var user string
	switch {
	case sshable.user != "":
		user = sshable.user
	case serverDetail.Hypervisor == comapi.HYPERVISOR_KVM:
		user = "root"
	default:
		user = "cloudroot"
	}
	// create local forward
	createP := jsonutils.NewDict()
	createP.Set("type", jsonutils.NewString(cloudproxy_api.FORWARD_TYPE_LOCAL))
	createP.Set("remote_port", jsonutils.NewInt(22))
	createP.Set("server_id", jsonutils.NewString(serverDetail.Id))

	forward, err := cloudproxy.Forwards.PerformClassAction(session, "create-from-server", createP)
	if err != nil {
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "fail to create local forward from server %q", serverDetail.Id))
		return
	}

	port, _ := forward.Int("bind_port")
	forwardId, _ := forward.GetString("id")
	agentId, _ := forward.GetString("proxy_agent_id")
	agent, err := cloudproxy.ProxyAgents.Get(session, agentId, nil)
	if err != nil {
		self.clearLocalForward(session, forwardId)
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "fail to get proxy agent %q", agentId))
		return
	}
	address, _ := agent.GetString("advertise_addr")
	host := ansible_api.AnsibleHost{
		User: user,
		IP:   address,
		Port: int(port),
		Name: serverDetail.Name,
	}
	params = jsonutils.NewDict()
	params.Set("args", sa.Args)
	params.Set("host", jsonutils.Marshal(host))
	// fetch ansible playbook reference id
	updateData := jsonutils.NewDict()
	updateData.Set("script_apply_record_id", jsonutils.NewString(sar.GetId()))
	updateData.Set("proxy_forward_id", jsonutils.NewString(forwardId))

	// check proxy forward
	if ok := self.ensureLocalForwardWork(address, int(port)); !ok {
		self.clearLocalForward(session, forwardId)
		self.taskFailed(ctx, sa, sar, errors.Error("The created local forward is actually not usable"))
		return
	}
	self.SetStage("OnAnsiblePlaybookComplete", updateData)
	// Inject Task Header
	session.Header = self.GetTaskRequestHeader()
	_, err = modules.AnsiblePlaybookReference.PerformAction(session, s.PlaybookReferenceId, "run", params)
	if err != nil {
		self.clearLocalForward(session, forwardId)
		self.taskFailed(ctx, sa, sar, errors.Wrapf(err, "can't run ansible playbook reference %s", s.PlaybookReferenceId))
		return
	}
}

type sSSHable struct {
	user   string
	ok     bool
	reason string
}

func (self *ApplyScriptTask) checkSshable(session *mcclient.ClientSession, serverId string) (sSSHable, error) {
	data, err := modules.Servers.GetSpecific(session, serverId, "sshable", nil)
	if err != nil {
		return sSSHable{}, errors.Wrapf(err, "unable to get sshable info of server %s", serverId)
	}
	log.Debugf("data to chech sshable:\n %s", data)
	methodTrieds, _ := data.GetArray("method_tried")
	sshable := sSSHable{}
	reasons := make([]string, 0, len(methodTrieds))
	for _, methodTried := range methodTrieds {
		ok, _ := methodTried.Bool("sshable")
		if ok {
			sshable.ok = true
			break
		}
		reason, _ := methodTried.GetString("reason")
		reasons = append(reasons, reason)
	}
	if !sshable.ok {
		sshable.reason = strings.Join(reasons, "; ")
	} else {
		sshable.user, _ = data.GetString("user")
	}
	return sshable, nil
}

func (self *ApplyScriptTask) clearLocalForward(s *mcclient.ClientSession, forwardId string) {
	_, err := cloudproxy.Forwards.Delete(s, forwardId, nil)
	if err != nil {
		log.Errorf("unable to delete proxy forward %s", forwardId)
	}
}

func (self *ApplyScriptTask) ensureLocalForwardWork(host string, port int) bool {
	maxWaitTimes, wt := 10, 1*time.Second
	waitTimes := 1
	address := fmt.Sprintf("%s:%d", host, port)
	for waitTimes < maxWaitTimes {
		_, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			return true
		}
		log.Debugf("no.%d times, try to connect to %s failed: %s", waitTimes, address, err)
		time.Sleep(wt)
		waitTimes += 1
		wt += 1 * time.Second
	}
	return false
}

func mapStringSlice(f func(string) string, a []string) []string {
	for i := range a {
		a[i] = f(a[i])
	}
	return a
}

const (
	agentInstalledKey   = "sys:monitor_agent"
	agentInstalledValue = "true"
)

func (self *ApplyScriptTask) OnAnsiblePlaybookComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	// try to delete local forward
	session := auth.GetAdminSession(ctx, "", "")
	forwardId, _ := self.Params.GetString("proxy_forward_id")
	self.clearLocalForward(session, forwardId)

	sa := obj.(*models.SScriptApply)
	// try to set metadata for guest
	metadata := jsonutils.NewDict()
	metadata.Set(agentInstalledKey, jsonutils.NewString(agentInstalledValue))
	_, err := modules.Servers.PerformAction(session, sa.GuestId, "metadata", metadata)
	if err != nil {
		log.Errorf("set metadata '%s:%s' for guest %s failed: %v", agentInstalledKey, agentInstalledValue, sa.GuestId, err)
	}

	sarId, _ := self.Params.GetString("script_apply_record_id")
	osar, err := models.ScriptApplyRecordManager.FetchById(sarId)
	if err != nil {
		log.Errorf("unable to fetch script apply record %s", sarId)
		self.taskSuccess(ctx, sa, nil)
	}
	self.taskSuccess(ctx, sa, osar.(*models.SScriptApplyRecord))
}

func (self *ApplyScriptTask) OnAnsiblePlaybookCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	// try to delete local forward
	session := auth.GetAdminSession(ctx, "", "")
	forwardId, _ := self.Params.GetString("proxy_forward_id")
	_, err := cloudproxy.Forwards.Delete(session, forwardId, nil)
	if err != nil {
		log.Errorf("unable to delete proxy forward %s", forwardId)
	}
	sa := obj.(*models.SScriptApply)
	sarId, _ := self.Params.GetString("script_apply_record_id")
	osar, err := models.ScriptApplyRecordManager.FetchById(sarId)
	if err != nil {
		log.Errorf("unable to fetch script apply record %s", sarId)
		self.taskSuccess(ctx, sa, nil)
	}
	self.taskFailed(ctx, sa, osar.(*models.SScriptApplyRecord), errors.Error(body.String()))
}
