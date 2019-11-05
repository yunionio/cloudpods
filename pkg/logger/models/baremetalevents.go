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

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SBaremetalEventManager struct {
	db.SModelBaseManager
}

type SBaremetalEvent struct {
	db.SModelBase

	Id       int64     `primary:"true" auto_increment:"true" list:"user"`
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

func init() {
	BaremetalEventManager = &SBaremetalEventManager{
		SModelBaseManager: db.NewModelBaseManager(
			SBaremetalEvent{},
			"baremetal_event_tbl",
			"baremetalevent",
			"baremetalevents",
		),
	}
	BaremetalEventManager.SetVirtualObject(BaremetalEventManager)
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

func (manager *SBaremetalEventManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SBaremetalEventManager) GetPagingConfig() *db.SPagingConfig {
	return &db.SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerField:  "id",
		DefaultLimit: 20,
	}
}

func (manager *SBaremetalEventManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	since, _ := query.GetTime("since")
	if !since.IsZero() {
		q = q.GT("created", since)
	}
	until, _ := query.GetTime("until")
	if !until.IsZero() {
		q = q.LE("created", until)
	}
	return q, nil
}
