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
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSshInfo struct {
	db.SStatusStandaloneResourceBase
	ServerId         string `width:"128" charset:"ascii" list:"user" create:"required"`
	ServerName       string `width:"128" charset:"utf8" list:"user" create:"optional"`
	ServerHypervisor string `width:"16" charset:"ascii" create:"optional"`
	ForwardId        string `width:"128" charset:"ascii" create:"optional"`
	User             string `width:"36" list:"user" create:"optional"`
	Host             string `width:"36" charset:"ascii" list:"user" create:"optional"`
	Port             int    `width:"8" charset:"ascii" list:"user" create:"optional"`
	NeedClean        tristate.TriState
	FailedReason     string
}

type SSshInfoManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var SshInfoManager *SSshInfoManager

func init() {
	SshInfoManager = &SSshInfoManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SSshInfo{},
			"sshinfo_tbl",
			"sshinfo",
			"sshinfos",
		),
	}
	SshInfoManager.SetVirtualObject(SshInfoManager)
}

func (si *SSshInfo) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	si.SetStatus(ctx, userCred, api.SSHINFO_STATUS_CREATING, "")

	task, err := taskman.TaskManager.NewTask(ctx, "SshInfoCreateTask", si, userCred, nil, "", "")
	if err != nil {
		log.Errorf("start SshInfoCreateTask failed: %v", err)
	}
	task.ScheduleRun(nil)
}

func (si *SSshInfo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (si *SSshInfo) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return si.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (si *SSshInfo) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	si.SetStatus(ctx, userCred, api.SSHINFO_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SshInfoDeleteTask", si, userCred, nil, "", "")
	if err != nil {
		log.Errorf("start SshInfoDeleteTask failed: %v", err)
	}
	task.ScheduleRun(nil)
	return nil
}

func (si *SSshInfo) MarkCreateFailed(reason string) {
	_, err := db.Update(si, func() error {
		si.Status = api.SSHINFO_STATUS_CREATE_FAILED
		si.FailedReason = reason
		return nil
	})
	if err != nil {
		log.Errorf("unable to mark createfailed for sshinfo: %v", err)
	}
}

func (si *SSshInfo) MarkDeleteFailed(reason string) {
	_, err := db.Update(si, func() error {
		si.Status = api.SSHINFO_STATUS_DELETE_FAILED
		si.FailedReason = reason
		return nil
	})
	if err != nil {
		log.Errorf("unable to mark deletefailed for sshinfo: %v", err)
	}
}
