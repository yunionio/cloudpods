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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNetworkAddressManager struct {
	db.SStandaloneAnonResourceBaseManager
	SNetworkResourceBaseManager
}

var NetworkAddressManager *SNetworkAddressManager

func init() {
	NetworkAddressManager = &SNetworkAddressManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SNetworkAddress{},
			"networkaddresses_tbl",
			"networkaddress",
			"networkaddresses",
		),
	}
	NetworkAddressManager.SetVirtualObject(NetworkAddressManager)
}

type SNetworkAddress struct {
	db.SStandaloneAnonResourceBase

	Type       string `width:"16" charset:"ascii" list:"user" create:"required"`
	ParentType string `width:"16" charset:"ascii" list:"user" create:"required"`
	ParentId   string `width:"16" charset:"ascii" list:"user" create:"optional"`
	SNetworkResourceBase
	IpAddr string `width:"16" charset:"ascii" list:"user" create:"optional"`

	SubCtrVid int `create:"optional"`
}

func (man *SNetworkAddressManager) queryByParentTypeId(ctx context.Context, typ string, id string) *sqlchemy.SQuery {
	q := NetworkAddressManager.Query().
		Equals("parent_type", typ).
		Equals("parent_id", id)
	return q
}

func (man *SNetworkAddressManager) queryByGuestnetworkId(ctx context.Context, rowid int64) *sqlchemy.SQuery {
	id := strconv.FormatInt(rowid, 10)
	return man.queryByParentTypeId(ctx, api.NetworkAddressParentTypeGuestnetwork, id)
}

func (man *SNetworkAddressManager) fetchByParentTypeId(ctx context.Context, typ string, id string) ([]SNetworkAddress, error) {
	var (
		q   = man.queryByParentTypeId(ctx, typ, id)
		nas []SNetworkAddress
	)
	if err := db.FetchModelObjects(man, q, &nas); err != nil {
		return nil, err
	}
	return nas, nil
}

func (man *SNetworkAddressManager) fetchByGuestnetworkId(ctx context.Context, rowid int64) ([]SNetworkAddress, error) {
	id := strconv.FormatInt(rowid, 10)
	return man.fetchByParentTypeId(ctx, api.NetworkAddressParentTypeGuestnetwork, id)
}

type addrConf struct {
	IpAddr  string `json:"ip_addr"`
	Masklen int    `json:"masklen"`
	Gateway string `json:"gateway"`
}

func (man *SNetworkAddressManager) fetchAddressesByGuestnetworkId(ctx context.Context, rowid int64) ([]addrConf, error) {
	var (
		naq     = man.queryByGuestnetworkId(ctx, rowid).SubQuery()
		nq      = NetworkManager.Query().SubQuery()
		ipnetsq = naq.Query(
			naq.Field("ip_addr"),
			nq.Field("guest_ip_mask").Label("masklen"),
			nq.Field("guest_gateway").Label("gateway"),
		).Join(nq, sqlchemy.Equals(
			naq.Field("network_id"),
			nq.Field("id")),
		)
		ipnets []addrConf
	)
	if err := ipnetsq.All(&ipnets); err != nil {
		return nil, errors.Wrapf(err, "fetch addresses ipnets by guestnetwork row id: %d", rowid)
	}
	return ipnets, nil
}

func (man *SNetworkAddressManager) deleteByGuestnetworkId(ctx context.Context, userCred mcclient.TokenCredential, rowid int64) error {
	nas, err := NetworkAddressManager.fetchByGuestnetworkId(ctx, rowid)
	if err != nil {
		return errors.Wrap(err, "fetch attached network addresses")
	}
	var errs []error
	for i := range nas {
		na := &nas[i]
		err := db.DeleteModel(ctx, userCred, na)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SNetworkAddressManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var (
		input api.NetworkAddressCreateInput
		err   error
	)
	if data, err = man.SStandaloneAnonResourceBaseManager.BatchCreateValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}
	if err = data.Unmarshal(&input); err != nil {
		return nil, err
	}
	input, err = man.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func (man *SNetworkAddressManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NetworkAddressCreateInput) (api.NetworkAddressCreateInput, error) {
	if _, err := man.SStandaloneAnonResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneAnonResourceCreateInput); err != nil {
		return input, err
	}
	var (
		net *SNetwork
	)
	switch input.ParentType {
	case api.NetworkAddressParentTypeGuestnetwork:
		switch input.Type {
		case api.NetworkAddressTypeSubIP:
			if input.GuestId != "" {
				gM, err := GuestManager.FetchByIdOrName(userCred, input.GuestId)
				if err != nil {
					return input, httperrors.NewInputParameterError("fetch guest %s: %v", input.GuestId, err)
				}
				gn, err := GuestnetworkManager.FetchByGuestIdIndex(gM.GetId(), input.GuestnetworkIndex)
				if err != nil {
					return input, httperrors.NewInputParameterError("fetch guest nic: %v", err)
				}
				net = gn.GetNetwork()
				if net == nil {
					return input, httperrors.NewInternalServerError("cannot fetch network of guestnetwork %d", gn.RowId)
				}
				input.ParentId = gn.RowId
				input.NetworkId = net.Id
			} else {
				return input, httperrors.NewInputParameterError("unknown parent object id spec")
			}

		default:
			return input, httperrors.NewInputParameterError("got unknown type %q, expect %s",
				input.Type, api.NetworkAddressTypes)
		}
	default:
		return input, httperrors.NewInputParameterError("got unknown parent type %q, expect %s",
			input.ParentType, api.NetworkAddressParentTypes)
	}

	if net == nil {
		panic("fatal error: no network")
	}
	{
		tryReserved := false
		if input.IPAddr != "" {
			tryReserved = true
		}
		ipAddr, err := net.GetFreeIPWithLock(ctx, userCred, nil, nil, input.IPAddr, "", tryReserved)
		if err != nil {
			return input, httperrors.NewInputParameterError("allocate ip addr: %v", err)
		}
		input.IPAddr = ipAddr
	}

	return input, nil
}

func (man *SNetworkAddressManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	return db.ApplyListItemExportKeys(ctx, q, userCred, keys,
		&man.SStandaloneAnonResourceBaseManager,
		&man.SNetworkResourceBaseManager,
	)
}

func (man *SNetworkAddressManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	return db.ApplyQueryDistinctExtraField(q, field,
		&man.SStandaloneAnonResourceBaseManager,
		&man.SNetworkResourceBaseManager,
	)
}

func (man *SNetworkAddressManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.NetworkAddressListInput) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = man.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, input.NetworkFilterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SNetworkAddressManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkAddressListInput,
) (retq *sqlchemy.SQuery, err error) {
	retq, err = man.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, err
	}
	retq, err = man.SStandaloneAnonResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	return retq, nil
}

func (man *SNetworkAddressManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkAddressDetails {
	ret := make([]api.NetworkAddressDetails, len(objs))
	netCols := man.SNetworkResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range netCols {
		ret[i].NetworkResourceInfo = netCols[i]
	}
	stdaCols := man.SStandaloneAnonResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range stdaCols {
		ret[i].StandaloneAnonResourceDetails = stdaCols[i]
	}

	{
		guestnetworkRowIds := make([]int64, 0, len(objs))
		for i := range objs {
			na := objs[i].(*SNetworkAddress)
			if na.ParentType == api.NetworkAddressParentTypeGuestnetwork {
				rowId, _ := strconv.ParseInt(na.ParentId, 10, 64)
				guestnetworkRowIds = append(guestnetworkRowIds, rowId)
			}
		}
		if len(guestnetworkRowIds) > 0 {
			gns, err := GuestnetworkManager.fetchByRowIds(ctx, userCred, guestnetworkRowIds)
			if err == nil && len(gns) > 0 {
				gnObjs := make([]interface{}, len(gns))
				for i := range gns {
					gnObjs[i] = &gns[i]
				}
				gnds := GuestnetworkManager.FetchCustomizeColumns(ctx, userCred, query, gnObjs, fields, isList)
				for i, j := 0, 0; i < len(objs); i++ {
					na := objs[i].(*SNetworkAddress)
					if na.ParentType == api.NetworkAddressParentTypeGuestnetwork {
						ret[i].Guestnetwork = gnds[j]
						j++
					}
				}
			}
		}
	}
	return ret
}

func (man *SNetworkAddressManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = db.ApplyFilterByOwner(q, owner, scope,
		&man.SStandaloneAnonResourceBaseManager,
	)
	if owner != nil {
		var condVar, condVal string
		switch scope {
		case rbacutils.ScopeProject:
			condVar, condVal = "tenant_id", owner.GetProjectId()
		case rbacutils.ScopeDomain:
			condVar, condVal = "domain_id", owner.GetProjectDomainId()
		default:
			return q
		}
		{
			var (
				gnq = GuestnetworkManager.Query().SubQuery()
				gq  = GuestManager.Query().SubQuery()
				naq = NetworkAddressManager.Query("id")
			)
			naq = naq.
				Join(gnq, sqlchemy.Equals(naq.Field("parent_id"), gnq.Field("row_id"))).
				Join(gq, sqlchemy.Equals(gnq.Field("guest_id"), gq.Field("id")))
			naq = naq.Equals("parent_type", api.NetworkAddressParentTypeGuestnetwork)
			naq = naq.Filter(sqlchemy.Equals(gq.Field(condVar), condVal))
			q = q.In("id", naq.SubQuery())
		}
	}
	return q
}
