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
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/sqlchemy/backends/clickhouse"

	api "yunion.io/x/onecloud/pkg/apis/logger"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaremetalEventManager struct {
	db.SModelBaseManager
}

type SBaremetalEvent struct {
	db.SModelBase

	Id       int64     `primary:"true" auto_increment:"true" list:"user" clickhouse_partition_by:"toInt64(id/100000000000)"`
	HostId   string    `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	HostName string    `width:"64" charset:"utf8" nullable:"false" list:"user" create:"required"`
	IpmiIp   string    `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Created  time.Time `nullable:"false" create:"required" list:"user"`
	EventId  string    `width:"32" nullable:"true" create:"optional" list:"user"`
	Type     string    `width:"10" nullable:"true" create:"optional" list:"user"`

	Message  string `nullable:"false" create:"required" list:"user"`
	Severity string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

var BaremetalEventManager *SBaremetalEventManager

func InitBaremetalEvent() {
	if consts.OpsLogWithClickhouse {
		BaremetalEventManager = &SBaremetalEventManager{
			SModelBaseManager: db.NewModelBaseManagerWithDBName(
				SBaremetalEvent{},
				"baremetal_event_tbl",
				"baremetalevent",
				"baremetalevents",
				db.ClickhouseDB,
			),
		}
		col := BaremetalEventManager.TableSpec().ColumnSpec("ops_time")
		if clickCol, ok := col.(clickhouse.IClickhouseColumnSpec); ok {
			clickCol.SetTTL(consts.SplitableMaxKeepMonths(), "MONTH")
		}
	} else {
		BaremetalEventManager = &SBaremetalEventManager{
			SModelBaseManager: db.NewModelBaseManagerWithSplitable(
				SBaremetalEvent{},
				"baremetal_event_tbl",
				"baremetalevent",
				"baremetalevents",
				"id",
				"created",
				consts.SplitableMaxDuration(),
				consts.SplitableMaxKeepMonths(),
			),
		}
	}
	BaremetalEventManager.SetVirtualObject(BaremetalEventManager)
}

func (event *SBaremetalEvent) BeforeInsert() {
	t := time.Now().UTC()
	event.Id = db.CurrentTimestamp(t)
}

func (event *SBaremetalEvent) GetId() string {
	return fmt.Sprintf("%d", event.Id)
}

func (event *SBaremetalEvent) GetName() string {
	return event.HostName + event.EventId
}

func (event *SBaremetalEvent) GetModelManager() db.IModelManager {
	return BaremetalEventManager
}

func (manager *SBaremetalEventManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SBaremetalEventManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SBaremetalEventManager) GetPagingConfig() *db.SPagingConfig {
	return &db.SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerFields: []string{"id"},
		DefaultLimit: 20,
	}
}

// 物理机日志列表
func (manager *SBaremetalEventManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BaremetalEventListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}

	if !query.Since.IsZero() {
		q = q.GT("created", query.Since)
	}
	if !query.Until.IsZero() {
		q = q.LE("created", query.Until)
	}
	if len(query.HostId) == 1 {
		q = q.Equals("host_id", query.HostId[0])
	} else if len(query.HostId) > 1 {
		q = q.In("host_id", query.HostId)
	}
	if len(query.Id) == 1 {
		q = q.Equals("id", query.Id[0])
	} else if len(query.Id) > 1 {
		q = q.In("id", query.Id)
	}
	if len(query.EventId) == 1 {
		q = q.Equals("event_id", query.EventId[0])
	} else if len(query.EventId) > 1 {
		q = q.In("event_id", query.EventId)
	}
	if len(query.Type) == 1 {
		q = q.Equals("type", query.Type[0])
	} else if len(query.Type) > 1 {
		q = q.In("type", query.Type)
	}
	if len(query.IpmiIp) == 1 {
		q = q.Equals("ipmi_ip", query.IpmiIp[0])
	} else if len(query.IpmiIp) > 1 {
		q = q.In("ipmi_ip", query.IpmiIp)
	}
	return q, nil
}
