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
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostDmesgLogManager struct {
	db.SOpsLogManager
}

type SHostDmesgLog struct {
	db.SOpsLog

	LogLevel string `width:"8" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

var HostDmesgLogManager *SHostDmesgLogManager

var _ db.IModelManager = (*SHostDmesgLogManager)(nil)
var _ db.IModel = (*SHostDmesgLog)(nil)

func NewHostDmesgLogManager(opslog interface{}, tblName string, keyword, keywordPlural string, timeField string, clickhouse bool) SHostDmesgLogManager {
	return SHostDmesgLogManager{
		SOpsLogManager: db.NewOpsLogManager(opslog, tblName, keyword, keywordPlural, timeField, clickhouse),
	}
}

func init() {
	tmp := NewHostDmesgLogManager(SHostDmesgLog{}, "hostdmesg_log_tbl", "hostdmesg", "hostdmesgs", "ops_time", consts.OpsLogWithClickhouse)
	HostDmesgLogManager = &tmp
	HostDmesgLogManager.SetVirtualObject(HostDmesgLogManager)
}

func (manager *SHostDmesgLogManager) LogDmesg(ctx context.Context, host *SHost, logLevel string, opsTime time.Time, notes interface{}, userCred mcclient.TokenCredential) {
	dmesgLog := &SHostDmesgLog{}
	dmesgLog.OpsTime = opsTime
	dmesgLog.LogLevel = logLevel
	dmesgLog.ObjId = host.GetId()
	dmesgLog.ObjName = host.GetName()
	dmesgLog.Action = db.ACT_HOST_DMESG
	dmesgLog.ObjType = host.Keyword()
	dmesgLog.Notes = stringutils.Interface2String(notes)
	dmesgLog.ProjectId = userCred.GetProjectId()
	dmesgLog.Project = userCred.GetProjectName()
	dmesgLog.ProjectDomainId = userCred.GetProjectDomainId()
	dmesgLog.ProjectDomain = userCred.GetProjectDomain()
	dmesgLog.UserId = userCred.GetUserId()
	dmesgLog.User = userCred.GetUserName()
	dmesgLog.DomainId = userCred.GetDomainId()
	dmesgLog.Domain = userCred.GetDomainName()
	dmesgLog.Roles = strings.Join(userCred.GetRoles(), ",")

	dmesgLog.SetModelManager(manager, dmesgLog)
	err := manager.TableSpec().Insert(ctx, dmesgLog)
	if err != nil {
		log.Errorf("fail to insert dmesgLog %s", err)
	}
}

func (manager *SHostDmesgLogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.HostDmesgLogListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SOpsLogManager.ListItemFilter(ctx, q, userCred, input.OpsLogListInput)
	if err != nil {
		return q, err
	}

	if len(input.LogLevels) > 0 {
		if len(input.LogLevels) == 1 {
			q = q.Filter(sqlchemy.Equals(q.Field("log_level"), input.LogLevels[0]))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("log_level"), input.LogLevels))
		}
	}

	return q, nil
}
