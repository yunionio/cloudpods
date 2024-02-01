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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SServiceUrl struct {
	db.SStatusStandaloneResourceBase
	Service           string              `width:"32" charset:"ascii" list:"user" create:"required"`
	ServerId          string              `width:"128" charset:"ascii" list:"user" create:"required"`
	Url               string              `wdith:"32" charset:"ascii" list:"user"`
	ServerAnsibleInfo *SServerAnisbleInfo `width:"128" list:"user" create:"required"`
	FailedReason      string
}

type SServiceUrlManager struct {
	db.SStatusStandaloneResourceBaseManager
}

type SServerAnisbleInfo struct {
	User string `json:"user"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Name string `json:"name"`
}

var ServiceUrlManager *SServiceUrlManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SServerAnisbleInfo{}), func() gotypes.ISerializable {
		return &SServerAnisbleInfo{}
	})
	ServiceUrlManager = &SServiceUrlManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SServiceUrl{},
			"serviceurl_tbl",
			"serviceurl",
			"serviceurls",
		),
	}
	ServiceUrlManager.SetVirtualObject(ServiceUrlManager)
}

func (ai *SServerAnisbleInfo) String() string {
	return jsonutils.Marshal(ai).String()
}

func (ai *SServerAnisbleInfo) IsZero() bool {
	return ai == nil
}

func (su *SServiceUrl) MarkCreateFailed(reason string) {
	_, err := db.Update(su, func() error {
		su.Status = api.SERVICEURL_STATUS_CREATE_FAILED
		su.FailedReason = reason
		return nil
	})
	if err != nil {
		log.Errorf("unable to mark createfailed for sshinfo: %v", err)
	}
}

func (su *SServiceUrl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	su.SetStatus(ctx, userCred, api.SERVICEURL_STATUS_CREATING, "")

	task, err := taskman.TaskManager.NewTask(ctx, "ServiceUrlCreateTask", su, userCred, nil, "", "")
	if err != nil {
		log.Errorf("start ServiceUrlCreateTask failed: %v", err)
	}
	task.ScheduleRun(nil)
}
