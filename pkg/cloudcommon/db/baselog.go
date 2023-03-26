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

package db

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/sqlchemy/backends/clickhouse"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLogBaseManager struct {
	SModelBaseManager
}

type SLogBase struct {
	SModelBase

	Id int64 `primary:"true" auto_increment:"true" list:"user" clickhouse_partition_by:"toInt64(id/100000000000)"`
}

func NewLogBaseManager(model interface{}, table string, keyword, keywordPlural string, timeCol string, useClickHouse bool) SLogBaseManager {
	if useClickHouse {
		man := SLogBaseManager{NewModelBaseManagerWithDBName(
			model,
			table,
			keyword,
			keywordPlural,
			ClickhouseDB,
		)}
		col := man.TableSpec().ColumnSpec("timeCol")
		if clickCol, ok := col.(clickhouse.IClickhouseColumnSpec); ok {
			clickCol.SetTTL(consts.SplitableMaxKeepMonths(), "MONTH")
		}
		return man
	} else {
		return SLogBaseManager{NewModelBaseManagerWithSplitable(
			model,
			table,
			keyword,
			keywordPlural,
			"id",
			timeCol,
			consts.SplitableMaxDuration(),
			consts.SplitableMaxKeepMonths(),
		)}
	}
}

func (manager *SLogBaseManager) CreateByInsertOrUpdate() bool {
	return false
}

func CurrentTimestamp(t time.Time) int64 {
	ret := int64(0)
	const (
		yOffset = 10000000000000
		mOffset = 100000000000
		dOffset = 1000000000
		hOffset = 10000000
		iOffset = 100000
		sOffset = 1000
	)
	ret += int64(t.Year()) * yOffset
	ret += int64(t.Month()) * mOffset
	ret += int64(t.Day()) * dOffset
	ret += int64(t.Hour()) * hOffset
	ret += int64(t.Minute()) * iOffset
	ret += int64(t.Second()) * sOffset
	ret += int64(t.Nanosecond()) / 1000000
	return ret
}

func (opslog *SLogBase) BeforeInsert() {
	t := time.Now().UTC()
	opslog.Id = CurrentTimestamp(t)
}

func (opslog *SLogBase) GetId() string {
	return fmt.Sprintf("%d", opslog.Id)
}

func (self *SLogBase) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return httperrors.NewForbiddenError("not allow to delete log")
}

func (self *SLogBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	id, _ := strconv.Atoi(idStr)
	return q.Equals("id", id)
}

func (self *SLogBaseManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	id, _ := strconv.Atoi(idStr)
	return q.NotEquals("id", id)
}

func (self *SLogBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q
}

func (manager *SLogBaseManager) GetPagingConfig() *SPagingConfig {
	return &SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerFields: []string{"id"},
		DefaultLimit: 20,
	}
}

func (lb *SLogBase) GetRecordTime() time.Time {
	log.Fatalf("not implemented yet!")
	return time.Time{}
}

func (manager *SLogBaseManager) FetchById(idStr string) (IModel, error) {
	return FetchById(manager.GetIModelManager(), idStr)
}

func (l *SLogBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(httperrors.ErrForbidden, "not allow")
}
