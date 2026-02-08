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

	"github.com/golang-plus/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	guest_api "yunion.io/x/onecloud/pkg/apis/compute/guest"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGuestNetworkTrafficLogManager struct {
	db.SLogBaseManager
	db.SProjectizedResourceBaseManager
	SGuestResourceBaseManager
	SNetworkResourceBaseManager
}

type SGuestNetworkTrafficLog struct {
	db.SLogBase
	db.SProjectizedResourceBase
	SGuestResourceBase
	SNetworkResourceBase

	// MAC地址
	Mac string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	// IPv4地址
	IpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user"`
	// IPv6地址
	Ip6Addr string `width:"64" charset:"ascii" nullable:"true" list:"user"`

	// 下行流量，单位 bytes
	RxBytes int64 `nullable:"false" default:"0" list:"user" old_name:"rx_traffic_used"`
	// 上行流量，单位 bytes
	TxBytes int64 `nullable:"false" default:"0" list:"user" old_name:"tx_traffic_used"`

	ReportAt time.Time `nullable:"false" list:"user"`

	State guest_api.GuestNetworkTrafficState `width:"10" charset:"ascii" nullable:"false" default:"" list:"user" json:"state"`
}

var GuestNetworkTrafficLogManager *SGuestNetworkTrafficLogManager

var _ db.IModelManager = (*SGuestNetworkTrafficLogManager)(nil)
var _ db.IModel = (*SGuestNetworkTrafficLog)(nil)

func InitGuestNetworkTrafficLog() {
	GuestNetworkTrafficLogManager = &SGuestNetworkTrafficLogManager{
		SLogBaseManager: db.NewLogBaseManager(SGuestNetworkTrafficLog{},
			"guest_network_traffic_log_tbl",
			"guest_network_traffic_log",
			"guest_network_traffic_logs",
			"report_at", consts.OpsLogWithClickhouse),
	}
	GuestNetworkTrafficLogManager.SetVirtualObject(GuestNetworkTrafficLogManager)
}

// 操作日志列表
func (manager *SGuestNetworkTrafficLogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input guest_api.GuestNetworkTrafficLogListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ServerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, input.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemFilter")
	}
	if len(input.IpAddr) > 0 {
		q = q.In("ip_addr", input.IpAddr)
	}
	if len(input.Ip6Addr) > 0 {
		q = q.In("ip6_addr", input.Ip6Addr)
	}
	if !input.Since.IsZero() {
		q = q.GT("ops_time", input.Since)
	}
	if !input.Until.IsZero() {
		q = q.LE("ops_time", input.Until)
	}
	if len(input.State) > 0 {
		q = q.In("state", input.State)
	}
	return q, nil
}

func (manager *SGuestNetworkTrafficLogManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SGuestNetworkTrafficLogManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.QueryDistinctExtraField")
	}
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.QueryDistinctExtraField")
	}
	q, err = manager.SProjectizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.QueryDistinctExtraField")
	}
	return q, nil
}

func (manager *SGuestNetworkTrafficLogManager) NamespaceScope() rbacscope.TRbacScope {
	return manager.SProjectizedResourceBaseManager.NamespaceScope()
}

func (manager *SGuestNetworkTrafficLogManager) logTraffic(ctx context.Context, guest *SGuest, gn *SGuestnetwork, matrics *api.SNicTrafficRecord, tm time.Time, isReset bool) error {
	logGen := func(tx, rx int64) *SGuestNetworkTrafficLog {
		log := &SGuestNetworkTrafficLog{}
		log.SetModelManager(manager, log)

		log.Mac = gn.MacAddr
		log.GuestId = guest.Id
		log.NetworkId = gn.NetworkId
		log.ProjectId = guest.ProjectId
		log.DomainId = guest.DomainId
		log.IpAddr = gn.IpAddr
		log.Ip6Addr = gn.Ip6Addr
		log.RxBytes = rx
		log.TxBytes = tx
		log.ReportAt = tm
		log.State = guest_api.GuestNetworkTrafficStateContinue

		return log
	}

	if matrics != nil {
		log := logGen(matrics.TxTraffic, matrics.RxTraffic)
		err := manager.TableSpec().Insert(ctx, log)
		if err != nil {
			return errors.Wrap(err, "insert guest network traffic log")
		}
	}

	if isReset {
		log := logGen(0, 0)
		log.State = guest_api.GuestNetworkTrafficStateStart
		err := manager.TableSpec().Insert(ctx, log)
		if err != nil {
			return errors.Wrap(err, "insert guest network traffic log")
		}
	}
	return nil
}

func (manager *SGuestNetworkTrafficLogManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query guest_api.GuestNetworkTrafficLogListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ServerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SProjectizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SGuestNetworkTrafficLogManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []guest_api.GuestNetworkTrafficLogDetails {
	rows := make([]guest_api.GuestNetworkTrafficLogDetails, len(objs))
	return rows
}
