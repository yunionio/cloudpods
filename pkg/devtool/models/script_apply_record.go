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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SScriptApplyRecord struct {
	db.SStatusStandaloneResourceBase
	ScriptApplyId string    `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true"`
	StartTime     time.Time `list:"user"`
	EndTime       time.Time `list:"user"`
	Reason        string    `list:"user"`
	FailCode      string    `list:"user"`
}

type SScriptApplyRecordManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var ScriptApplyRecordManager *SScriptApplyRecordManager

func init() {
	ScriptApplyRecordManager = &SScriptApplyRecordManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SScriptApplyRecord{},
			"scriptapplyrecord_tbl",
			"scriptapplyrecord",
			"scriptapplyrecords",
		),
	}
	ScriptApplyRecordManager.SetVirtualObject(ScriptApplyRecordManager)
}

func (sarm *SScriptApplyRecordManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ScriptApplyRecoredListInput) (*sqlchemy.SQuery, error) {
	q, err := sarm.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return q, err
	}
	if len(input.ScriptApplyId) > 0 {
		q = q.Equals("script_apply_id", input.ScriptApplyId)
	}
	if len(input.ScriptId) > 0 || len(input.ServerId) > 0 {
		saq := ScriptApplyManager.Query("id")
		if len(input.ScriptId) > 0 {
			saq = saq.Equals("script_id", input.ScriptId)
		}
		if len(input.ServerId) > 0 {
			saq = saq.Equals("guest_id", input.ServerId)
		}
		saqSub := saq.SubQuery()
		q = q.Join(saqSub, sqlchemy.Equals(q.Field("script_apply_id"), saqSub.Field("id")))
	}
	return q, nil
}

func (sarm *SScriptApplyRecordManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ScriptApplyRecordDetails {
	sDetails := sarm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	details := make([]api.ScriptApplyRecordDetails, len(objs))
	for i := range details {
		details[i].StandaloneResourceDetails = sDetails[i]
		scriptApplyRecord := objs[i].(*SScriptApplyRecord)
		sa, err := scriptApplyRecord.ScriptApply()
		if err != nil {
			log.Errorf("unable to get SScriptApply: %v", err)
		}
		details[i].ScriptId = sa.ScriptId
		details[i].ServerId = sa.GuestId
	}
	return details
}

func (sar *SScriptApplyRecord) ScriptApply() (*SScriptApply, error) {
	obj, err := ScriptApplyManager.FetchById(sar.ScriptApplyId)
	if err != nil {
		return nil, err
	}
	return obj.(*SScriptApply), nil
}

func (sarm *SScriptApplyRecordManager) CreateRecord(ctx context.Context, scriptApplyId string) (*SScriptApplyRecord, error) {
	return sarm.createRecordWithResult(ctx, scriptApplyId, nil, "")
}

func (sarm *SScriptApplyRecordManager) createRecordWithResult(ctx context.Context, scriptApplyId string, success *bool, reason string) (*SScriptApplyRecord, error) {
	now := time.Now()
	sar := &SScriptApplyRecord{
		StartTime:     now,
		ScriptApplyId: scriptApplyId,
	}
	if success == nil {
		sar.Status = api.SCRIPT_APPLY_RECORD_APPLYING
	} else if *success {
		sar.Status = api.SCRIPT_APPLY_RECORD_SUCCEED
	} else if !*success {
		sar.Status = api.SCRIPT_APPLY_RECORD_FAILED
	}
	sar.Reason = reason
	err := sarm.TableSpec().Insert(ctx, sar)
	if err != nil {
		return nil, err
	}
	sar.SetModelManager(sarm, sar)
	return sar, nil
}

func (sarm *SScriptApplyRecordManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (sarm *SScriptApplyRecordManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (sarm *SScriptApplyRecordManager) FileterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			scriptQ := ScriptManager.Query("id", "domain_id").SubQuery()
			q = q.Join(scriptQ, sqlchemy.Equals(q.Field("script_id"), scriptQ.Field("id")))
			q = q.Filter(sqlchemy.Equals(scriptQ.Field("domain_id"), owner.GetProjectDomainId()))
		}
	}
	return q
}

func (sarm *SScriptApplyRecordManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchDomainInfo(ctx, data)
}

func (sar *SScriptApplyRecord) GetOwnerId() mcclient.IIdentityProvider {
	obj, _ := ScriptApplyManager.FetchById(sar.ScriptApplyId)
	if obj == nil {
		return nil
	}
	return obj.GetOwnerId()
}

func (sar *SScriptApplyRecord) SetResult(status, failCode, reason string) error {
	_, err := db.Update(sar, func() error {
		sar.Status = status
		sar.Reason = reason
		sar.FailCode = failCode
		sar.EndTime = time.Now()
		return nil
	})
	return err
}

func (sar *SScriptApplyRecord) Fail(code string, reason string) error {
	return sar.SetResult(api.SCRIPT_APPLY_RECORD_FAILED, code, reason)
}

func (sar *SScriptApplyRecord) Succeed(reason string) error {
	return sar.SetResult(api.SCRIPT_APPLY_RECORD_SUCCEED, "", reason)
}
