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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafIPSetManager struct {
	db.SStatusInfrasResourceBaseManager
}

var WafIPSetManager *SWafIPSetManager

func init() {
	WafIPSetManager = &SWafIPSetManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SWafIPSet{},
			"waf_ipsets_tbl",
			"waf_ipset",
			"waf_ipsets",
		),
	}
	WafIPSetManager.SetVirtualObject(WafIPSetManager)
}

type SWafIPSet struct {
	db.SStatusInfrasResourceBase

	Addresses *cloudprovider.WafAddresses `list:"domain" update:"domain" create:"required"`
}

func (manager *SWafIPSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafIPSetDetails {
	rows := make([]api.WafIPSetDetails, len(objs))
	siRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafIPSetDetails{
			StatusInfrasResourceBaseDetails: siRows[i],
		}
	}
	return rows
}

// 列出WAF IPSets
func (manager *SWafIPSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SWafIPSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SWafIPSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SWafIPSetManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (self *SWafIPSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafIPSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SWafIPSet) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafIPSet) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafIPSetDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.WAF_IPSET_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafIPSet) GetCaches() ([]SWafIPSetCache, error) {
	q := WafIPSetCacheManager.Query().Equals("waf_ipset_id", self.Id)
	caches := []SWafIPSetCache{}
	err := db.FetchModelObjects(WafIPSetCacheManager, q, &caches)
	return caches, err
}
