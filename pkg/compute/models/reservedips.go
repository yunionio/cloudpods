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
	"database/sql"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SReservedipManager struct {
	db.SResourceBaseManager
	SNetworkResourceBaseManager
}

var ReservedipManager *SReservedipManager

func init() {
	ReservedipManager = &SReservedipManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SReservedip{},
			api.RESERVEDIP_TABLE,
			api.RESERVEDIP_RESOURCE_TYPE,
			api.RESERVEDIP_RESOURCE_TYPES,
		),
	}
	ReservedipManager.SetVirtualObject(ReservedipManager)
}

type SReservedip struct {
	db.SResourceBase
	SNetworkResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user"`

	// 自增Id
	Id int64 `primary:"true" auto_increment:"true" list:"user"`

	// IP子网Id
	// NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user"`

	// IP地址
	IpAddr string `width:"16" charset:"ascii" list:"user"`

	// IPv6地址
	Ip6Addr string `width:"64" charset:"ascii" list:"user"`

	// 预留原因或描述
	Notes string `width:"512" charset:"utf8" nullable:"true" list:"user" update:"user"`

	// 过期时间
	ExpiredAt time.Time `nullable:"true" list:"user"`

	// 状态
	Status string `width:"12" charset:"ascii" nullable:"false" default:"unknown" list:"user" create:"optional" update:"user"`
}

func (manager *SReservedipManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SReservedipManager) ReserveIP(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, ip string, notes string, addrType api.TAddressType) error {
	return manager.ReserveIPWithDuration(ctx, userCred, network, ip, notes, 0, addrType)
}

func (manager *SReservedipManager) ReserveIPWithDuration(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, ip string, notes string, duration time.Duration, addrType api.TAddressType) error {
	return manager.ReserveIPWithDurationAndStatus(ctx, userCred, network, ip, notes, duration, "", addrType)
}

func (manager *SReservedipManager) ReserveIPWithDurationAndStatus(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, ip string, notes string, duration time.Duration, status string, addrType api.TAddressType) error {
	expiredAt := time.Time{}
	if duration > 0 {
		expiredAt = time.Now().UTC().Add(duration)
	}
	rip := manager.getReservedIP(network, ip, addrType)
	if rip == nil {
		// not found
		rip = &SReservedip{
			Notes:     notes,
			ExpiredAt: expiredAt,
			Status:    status,
		}
		rip.SetModelManager(manager, rip)
		if addrType == api.AddressTypeIPv6 {
			rip.Ip6Addr = ip
		} else {
			rip.IpAddr = ip
		}
		rip.NetworkId = network.Id
		err := manager.TableSpec().Insert(ctx, rip)
		if err != nil {
			log.Errorf("ReserveIP fail: %s", err)
			return errors.Wrap(err, "Insert")
		}
	} else {
		// extend the expiration
		err := rip.extend(notes, expiredAt, status)
		if err != nil {
			return errors.Wrap(err, "extend")
		}
	}
	db.OpsLog.LogEvent(network, db.ACT_RESERVE_IP, rip.GetShortDesc(ctx), userCred)
	logclient.AddSimpleActionLog(network, logclient.ACT_RESERVE_IP, rip.GetShortDesc(ctx), userCred, true)
	return nil
}

func (rip *SReservedip) extendWithDuration(notes string, duration time.Duration, status string) error {
	expiredAt := time.Time{}
	if duration > 0 {
		expiredAt = time.Now().UTC().Add(duration)
	}
	return rip.extend(notes, expiredAt, status)
}

func (rip *SReservedip) extend(notes string, expiredAt time.Time, status string) error {
	_, err := db.Update(rip, func() error {
		rip.Notes = notes
		rip.ExpiredAt = expiredAt
		if len(status) > 0 {
			rip.Status = status
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (manager *SReservedipManager) getReservedIP(network *SNetwork, ip string, addrType api.TAddressType) *SReservedip {
	rip := SReservedip{}
	rip.SetModelManager(manager, &rip)

	q := manager.Query().Equals("network_id", network.Id)
	if addrType == api.AddressTypeIPv6 {
		q = q.Equals("ip6_addr", ip)
	} else {
		q = q.Equals("ip_addr", ip)
	}
	err := q.First(&rip)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("GetReservedIP fail: %s", err)
		}
		return nil
	}
	return &rip
}

func (manager *SReservedipManager) GetReservedIP(network *SNetwork, ip string, addrType api.TAddressType) *SReservedip {
	rip := manager.getReservedIP(network, ip, addrType)
	if rip == nil {
		log.Errorf("GetReserved IP %s: not found", ip)
		return nil
	}
	if rip.IsExpired() {
		log.Errorf("GetReserved IP %s: expired", ip)
		return nil
	}
	return rip
}

func (manager *SReservedipManager) GetReservedIPs(network *SNetwork) []SReservedip {
	rips := make([]SReservedip, 0)
	q := manager.Query().Equals("network_id", network.Id)
	q = filterExpiredReservedIps(q)
	err := db.FetchModelObjects(manager, q, &rips)
	if err != nil {
		log.Errorf("GetReservedIPs fail: %s", err)
		return nil
	}
	return rips
}

func (self *SReservedip) GetNetwork() *SNetwork {
	net, _ := NetworkManager.FetchById(self.NetworkId)
	if net != nil {
		return net.(*SNetwork)
	}
	return nil
}

func (rip *SReservedip) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := rip.SResourceBase.GetShortDesc(ctx)
	if len(rip.IpAddr) > 0 {
		desc.Set("ip_addr", jsonutils.NewString(rip.IpAddr))
	}
	if len(rip.Ip6Addr) > 0 {
		desc.Set("ip6_addr", jsonutils.NewString(rip.Ip6Addr))
	}
	return desc
}

func (self *SReservedip) Release(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork) error {
	// db.DeleteModel(self.ResourceModelManager(), self)
	if network == nil {
		network = self.GetNetwork()
	}
	err := db.DeleteModel(ctx, userCred, self)
	if err == nil && network != nil {
		db.OpsLog.LogEvent(network, db.ACT_RELEASE_IP, self.GetShortDesc(ctx), userCred)
		logclient.AddSimpleActionLog(network, logclient.ACT_RELEASE_IP, self.GetShortDesc(ctx), userCred, true)
	}
	return err
}

func (manager *SReservedipManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ReservedipDetails {
	rows := make([]api.ReservedipDetails, len(objs))

	resRows := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netRows := manager.SNetworkResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ReservedipDetails{
			ResourceBaseDetails: resRows[i],
			NetworkResourceInfo: netRows[i],
		}
		rows[i].Expired = objs[i].(*SReservedip).IsExpired()
	}

	return rows
}

// 预留IP地址列表
func (manager *SReservedipManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ReservedipListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if query.All == nil || *query.All == false {
		q = filterExpiredReservedIps(q)
	}

	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}
	if len(query.Status) > 0 {
		q = q.In("status", query.Status)
	}

	return q, nil
}

func filterExpiredReservedIps(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	return q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("expired_at")),
		sqlchemy.GT(q.Field("expired_at"), time.Now().UTC()),
	))
}

func filterExpiredReservedIp4s(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	return q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("expired_at")),
		sqlchemy.GT(q.Field("expired_at"), time.Now().UTC()),
	)).IsNotEmpty("ip_addr")
}

func filterExpiredReservedIp6s(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	return q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("expired_at")),
		sqlchemy.GT(q.Field("expired_at"), time.Now().UTC()),
	)).IsNotEmpty("ip6_addr")
}

func (manager *SReservedipManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ReservedipListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SReservedipManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (rip *SReservedip) GetId() string {
	return strconv.FormatInt(rip.Id, 10)
}

func (rip *SReservedip) GetName() string {
	return rip.GetId()
}

func (manager *SReservedipManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SReservedipManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.Equals("id", name)
}

func (rip *SReservedip) IsExpired() bool {
	if !rip.ExpiredAt.IsZero() && rip.ExpiredAt.Before(time.Now()) {
		return true
	}
	return false
}

func (self *SReservedip) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"network_id": self.NetworkId})
}

func (manager *SReservedipManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	networkId, _ := data.GetString("network_id")
	return jsonutils.Marshal(map[string]string{"network_id": networkId})
}

func (manager *SReservedipManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	networkId, _ := values.GetString("network_id")
	if len(networkId) > 0 {
		q = q.Equals("network_id", networkId)
	}
	return q
}

func (manager *SReservedipManager) NamespaceScope() rbacscope.TRbacScope {
	if consts.IsDomainizedNamespace() {
		return rbacscope.ScopeDomain
	} else {
		return rbacscope.ScopeSystem
	}
}

func (manager *SReservedipManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SReservedipManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			netsQ := NetworkManager.Query("id")
			netsQ = NetworkManager.FilterByOwner(ctx, netsQ, NetworkManager, userCred, owner, scope)
			netsSQ := netsQ.SubQuery()
			q = q.Join(netsSQ, sqlchemy.Equals(q.Field("network_id"), netsSQ.Field("id")))
		}
	}
	return q
}

func (manager *SReservedipManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchProjectInfo(ctx, data)
}

func (rip *SReservedip) GetOwnerId() mcclient.IIdentityProvider {
	network := rip.GetNetwork()
	if network != nil {
		return network.GetOwnerId()
	}
	return nil
}

func (manager *SReservedipManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}
	return manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
}
