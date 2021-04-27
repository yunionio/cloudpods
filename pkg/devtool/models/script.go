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
	"yunion.io/x/pkg/errors"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/devtool/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SScript struct {
	db.SSharableVirtualResourceBase
	// remote
	Type                string `width:"16" nullable:"false"`
	PlaybookReferenceId string `width:"128" nullable:"false"`
	MaxTryTimes         int    `default:"1"`
}

type SScriptManager struct {
	db.SSharableVirtualResourceBaseManager
}

var ScriptManager *SScriptManager

func init() {
	ScriptManager = &SScriptManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SScript{},
			"script_tbl",
			"script",
			"scripts",
		),
	}
	ScriptManager.SetVirtualObject(ScriptManager)
	utils.RegisterArgGenerator(MonitorAgent, utils.GetArgs)
}

var MonitorAgent = "monitor agent"

func (sm *SScriptManager) InitializeData() error {
	q := sm.Query().Equals("playbook_reference", MonitorAgent)
	n, err := q.CountWithError()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	s := SScript{
		PlaybookReferenceId: MonitorAgent,
	}
	s.ProjectId = "system"
	s.IsPublic = true
	s.PublicScope = "system"
	err = sm.TableSpec().Insert(context.Background(), &s)
	if err != nil {
		return err
	}
	return nil
}

func (sm *SScriptManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScriptCreateInput) (api.ScriptCreateInput, error) {
	// check ansible playbook reference
	session := auth.GetSessionWithInternal(ctx, userCred, "", "")
	pr, err := modules.AnsiblePlaybookReference.Get(session, input.PlaybookReference, nil)
	if err != nil {
		return input, errors.Wrapf(err, "unable to get AnsiblePlaybookReference %q", input.PlaybookReference)
	}
	id, _ := pr.GetString("id")
	input.PlaybookReference = id
	return input, nil
}

func (s *SScript) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	s.Status = api.SCRIPT_STATUS_READY
	s.PlaybookReferenceId, _ = data.GetString("playbook_reference")
	return nil
}

func (sm *SScriptManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ScriptDetails {
	vDetails := sm.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	details := make([]api.ScriptDetails, len(objs))
	for i := range details {
		details[i].SharableVirtualResourceDetails = vDetails[i]
		script := objs[i].(*SScript)
		ais, err := script.ApplyInfos()
		if err != nil {
			log.Errorf("unable to get ApplyInfos of script %s: %v", script.Id, err)
		}
		details[i].ApplyInfos = ais
	}
	return details
}

func (s *SScript) ApplyInfos() ([]api.SApplyInfo, error) {
	q := ScriptApplyManager.Query().Equals("script_id", s.Id)
	var sa []SScriptApply
	err := db.FetchModelObjects(ScriptApplyManager, q, &sa)
	if err != nil {
		return nil, err
	}
	ai := make([]api.SApplyInfo, len(sa))
	for i := range ai {
		ai[i].ServerId = sa[i].GuestId
		ai[i].TryTimes = sa[i].TryTimes
	}
	return ai, nil
}

func (s *SScript) AllowPerformApply(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return s.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, s, "apply")
}

func (s *SScript) PerformApply(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ScriptApplyInput) (api.ScriptApplyOutput, error) {
	output := api.ScriptApplyOutput{}
	var argsGenerator string
	if s.Name == MonitorAgent {
		argsGenerator = MonitorAgent
	}
	sa, err := ScriptApplyManager.createScriptApply(ctx, s.Id, input.ServerID, nil, argsGenerator)
	if err != nil {
		return output, errors.Wrapf(err, "unable to apply script to server %s", input.ServerID)
	}
	err = sa.StartApply(ctx, userCred)
	if err != nil {
		return output, errors.Wrapf(err, "unable to apply script to server %s", input.ServerID)
	}
	output.ScriptApplyId = sa.Id
	return output, nil
}

type sServerInfo struct {
	ServerId      string
	VpcId         string
	NetworkIds    []string
	serverDetails *comapi.ServerDetails
}
