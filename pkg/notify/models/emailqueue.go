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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/sender"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SEmailQueueManager struct {
	db.SLogBaseManager
}

type SEmailQueue struct {
	db.SLogBase

	RecvAt time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user" json:"recv_at"`

	Dest    string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`
	Subject string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"admin_required"`

	SessionId string `width:"256" charset:"utf8" nullable:"false" list:"user" create:"admin_optional"`

	Content jsonutils.JSONObject `length:"long" charset:"utf8" nullable:"false" list:"user" create:"admin_required"`

	ProjectId string `width:"128" charset:"ascii" list:"user" create:"admin_optional" index:"true"`
	Project   string `width:"128" charset:"utf8" list:"user" create:"admin_optional"`

	ProjectDomainId string `name:"project_domain_id" default:"default" width:"128" charset:"ascii" list:"user" create:"admin_optional"`
	ProjectDomain   string `name:"project_domain" default:"Default" width:"128" charset:"utf8" list:"user" create:"admin_optional"`

	UserId   string `width:"128" charset:"ascii" list:"user" create:"admin_required"`
	User     string `width:"128" charset:"utf8" list:"user" create:"admin_required"`
	DomainId string `width:"128" charset:"ascii" list:"user" create:"admin_optional"`
	Domain   string `width:"128" charset:"utf8" list:"user" create:"admin_optional"`
	Roles    string `width:"64" charset:"utf8" list:"user" create:"admin_optional"`
}

var EmailQueueManager *SEmailQueueManager

func InitEmailQueue() {
	EmailQueueManager = &SEmailQueueManager{
		SLogBaseManager: db.NewLogBaseManager(SEmailQueue{}, "emailqueue_tbl", "emailqueue", "emailqueues", "recv_at", consts.OpsLogWithClickhouse),
	}
	EmailQueueManager.SetVirtualObject(EmailQueueManager)
}

func (e *SEmailQueue) GetRecordTime() time.Time {
	return e.RecvAt
}

func (manager *SEmailQueueManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.EmailQueueCreateInput,
) (api.EmailQueueCreateInput, error) {
	// check permission
	if db.IsAdminAllowCreate(userCred, manager).Result.IsDeny() {
		return input, errors.Wrap(httperrors.ErrForbidden, "only admin can send email")
	}
	// validate data
	if len(input.To) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "empty receiver")
	}
	invalidTos := make([]string, 0)
	for _, to := range input.To {
		if !regutils.MatchEmail(to) {
			invalidTos = append(invalidTos, to)
		}
	}
	if len(invalidTos) > 0 {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid email %s", strings.Join(invalidTos, ","))
	}
	input.Dest = strings.Join(input.To, ",")
	msg := api.SEmailMessage{
		Body:        input.Body,
		Attachments: input.Attachments,
	}
	input.Content = jsonutils.Marshal(msg)

	input.Project = userCred.GetProjectName()
	input.ProjectId = userCred.GetProjectId()
	input.ProjectDomain = userCred.GetProjectDomain()
	input.ProjectDomainId = userCred.GetProjectDomainId()
	input.User = userCred.GetUserName()
	input.UserId = userCred.GetUserId()
	input.Domain = userCred.GetDomainName()
	input.DomainId = userCred.GetDomainId()
	input.Roles = strings.Join(userCred.GetRoles(), ",")

	return input, nil
}

func (eq *SEmailQueue) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	eq.SLogBase.PostCreate(ctx, userCred, ownerId, query, data)
	eq.setStatus(ctx, api.EmailQueued, nil)
	eq.doSendAsync()
}

func (eq *SEmailQueue) doSendAsync() {
	sender.Worker.Run(eq, nil, nil)
}

func (eq *SEmailQueue) Dump() string {
	return fmt.Sprintf("send email %s", eq.Subject)
}

func (eq *SEmailQueue) Run() {
	log.Debugf("send email")
	eq.doSend(context.TODO())
}

func (eq *SEmailQueue) doSend(ctx context.Context) {
	conf, err := ConfigManager.getEmailConfig()
	if err != nil {
		eq.setStatus(ctx, api.EmailFail, err)
		return
	}
	log.Debugf("conf: %s", jsonutils.Marshal(conf))
	msg, err := eq.getMessage()
	if err != nil {
		eq.setStatus(ctx, api.EmailFail, err)
		return
	}
	log.Debugf("msg: %s", jsonutils.Marshal(msg))
	eq.setStatus(ctx, api.EmailSending, nil)
	err = sender.SendEmail(conf, msg)
	if err != nil {
		eq.setStatus(ctx, api.EmailFail, err)
		return
	}
	eq.setStatus(ctx, api.EmailSuccess, nil)
	return
}

func (eq *SEmailQueue) getMessage() (*api.SEmailMessage, error) {
	msg := api.SEmailMessage{}
	err := eq.Content.Unmarshal(&msg)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	msg.To = strings.Split(eq.Dest, ",")
	msg.Subject = eq.Subject
	return &msg, nil
}

func (eq *SEmailQueue) setStatus(ctx context.Context, status string, results error) {
	eqs := SEmailQueueStatus{
		Id:     eq.Id,
		Status: status,
	}
	if results != nil {
		eqs.Results = results.Error()
	}
	if status == api.EmailSuccess || status == api.EmailFail {
		eqs.SentAt = time.Now()
	}
	EmailQueueStatusManager.TableSpec().InsertOrUpdate(ctx, &eqs)
}

// 宿主机/物理机列表
func (manager *SEmailQueueManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.EmailQueueListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SLogBaseManager.ListItemFilter(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SLogBaseManager.ListItemFilter")
	}

	if len(query.Id) > 0 {
		q = q.In("id", query.Id)
	}
	if len(query.To) > 0 {
		cond := make([]sqlchemy.ICondition, 0)
		for _, to := range query.To {
			cond = append(cond, sqlchemy.Contains(q.Field("dest"), to))
		}
		q = q.Filter(sqlchemy.OR(cond...))
	}
	if len(query.Subject) > 0 {
		q = q.Contains("subject", query.Subject)
	}
	if len(query.SessionId) > 0 {
		q = q.In("session_id", query.SessionId)
	}

	return q, nil
}

func (eq *SEmailQueue) PerformSend(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.EmailQueueSendInput,
) (jsonutils.JSONObject, error) {
	eq.setStatus(ctx, api.EmailQueued, nil)
	if input.Sync {
		log.Debugf("send email synchronously")
		eq.doSend(ctx)
	} else {
		log.Debugf("send email Asynchronously")
		eq.doSendAsync()
	}
	return nil, nil
}

func (manager *SEmailQueueManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.EmailQueueDetails {
	rows := make([]api.EmailQueueDetails, len(objs))

	baseRows := manager.SModelBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	emailIds := make([]int64, len(objs))
	for i := range rows {
		rows[i] = api.EmailQueueDetails{
			ModelBaseDetails: baseRows[i],
		}
		eq := objs[i].(*SEmailQueue)
		emailIds[i] = eq.Id
	}

	rets, err := EmailQueueStatusManager.fetchEmailQueueStatus(emailIds)
	if err != nil {
		log.Errorf("EmailQueueStatusManager.fetchEmailQueueStatus fail %s", err)
		return rows
	}

	for i := range rows {
		eq := objs[i].(*SEmailQueue)
		if eqs, ok := rets[eq.Id]; ok {
			rows[i].Status = eqs.Status
			rows[i].SentAt = eqs.SentAt
			rows[i].Results = eqs.Results
		}
	}

	return rows
}
