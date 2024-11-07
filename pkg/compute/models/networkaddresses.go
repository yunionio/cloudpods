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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/delayedwork"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=networkaddress
// +onecloud:swagger-gen-model-plural=networkaddresses
type SNetworkAddressManager struct {
	db.SStandaloneAnonResourceBaseManager
	SNetworkResourceBaseManager

	delayedWorkManager *delayedwork.DelayedWorkManager
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
	NetworkAddressManager.delayedWorkManager = delayedwork.NewDelayedWorkManager()
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

func (man *SNetworkAddressManager) InitializeData() error {
	go man.delayedWorkManager.Start(context.Background())
	return nil
}

func (man *SNetworkAddressManager) queryByParentTypeId(typ string, id string) *sqlchemy.SQuery {
	q := NetworkAddressManager.Query().
		Equals("parent_type", typ).
		Equals("parent_id", id)
	return q
}

func (man *SNetworkAddressManager) queryByGuestnetworkId(rowid int64) *sqlchemy.SQuery {
	id := strconv.FormatInt(rowid, 10)
	return man.queryByParentTypeId(api.NetworkAddressParentTypeGuestnetwork, id)
}

func (man *SNetworkAddressManager) fetchByParentTypeId(typ string, id string) ([]SNetworkAddress, error) {
	var (
		q   = man.queryByParentTypeId(typ, id)
		nas []SNetworkAddress
	)
	if err := db.FetchModelObjects(man, q, &nas); err != nil {
		return nil, err
	}
	return nas, nil
}

func (man *SNetworkAddressManager) fetchByGuestnetworkId(rowid int64) ([]SNetworkAddress, error) {
	id := strconv.FormatInt(rowid, 10)
	return man.fetchByParentTypeId(api.NetworkAddressParentTypeGuestnetwork, id)
}

func (man *SNetworkAddressManager) fetchAddressCountByGuestnetworkId(rowid int64) (int, error) {
	q := man.queryByGuestnetworkId(rowid)
	return q.CountWithError()
}

func (man *SNetworkAddressManager) fetchAddressesByGuestnetworkId(rowid int64) ([]api.NetworkAddrConf, error) {
	var (
		naq     = man.queryByGuestnetworkId(rowid).SubQuery()
		nq      = NetworkManager.Query().SubQuery()
		ipnetsq = naq.Query(
			naq.Field("id"),
			naq.Field("type"),
			naq.Field("ip_addr"),
			nq.Field("guest_ip_mask").Label("masklen"),
			nq.Field("guest_gateway").Label("gateway"),
		).Join(nq, sqlchemy.Equals(
			naq.Field("network_id"),
			nq.Field("id")),
		)
		ipnets []api.NetworkAddrConf
	)
	if err := ipnetsq.All(&ipnets); err != nil {
		return nil, errors.Wrapf(err, "fetch addresses ipnets by guestnetwork row id: %d", rowid)
	}
	return ipnets, nil
}

func (man *SNetworkAddressManager) deleteByGuestnetworkId(ctx context.Context, userCred mcclient.TokenCredential, rowid int64) error {
	nas, err := NetworkAddressManager.fetchByGuestnetworkId(rowid)
	if err != nil {
		return errors.Wrap(err, "fetch attached network addresses")
	}
	var errs []error
	for i := range nas {
		na := &nas[i]
		if err := na.remoteUnassignAddress(ctx, userCred); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := db.DeleteModel(ctx, userCred, na); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SNetworkAddressManager) syncGuestnetworkICloudNic(ctx context.Context, userCred mcclient.TokenCredential, guestnetwork *SGuestnetwork, iNic cloudprovider.ICloudNic) error {
	ipAddrs, err := iNic.GetSubAddress()
	if err != nil {
		return errors.Wrap(err, "iNic.GetSubAddress")
	}
	if err := man.syncGuestnetworkSubIPs(ctx, userCred, guestnetwork, ipAddrs); err != nil {
		return errors.Wrap(err, "syncGuestnetworkSubIPs")
	}
	return nil
}

func (man *SNetworkAddressManager) syncGuestnetworkSubIPs(ctx context.Context, userCred mcclient.TokenCredential, guestnetwork *SGuestnetwork, ipAddrs []string) error {
	nas, err := man.fetchByGuestnetworkId(guestnetwork.RowId)
	if err != nil {
		return errors.Wrap(err, "fetchByGuestnetworkId")
	}

	existings := stringutils2.NewSortedStrings(nil)
	for i := range nas {
		existings = existings.Append(nas[i].IpAddr)
	}
	probed := stringutils2.NewSortedStrings(ipAddrs)

	removes, _, adds := stringutils2.Split(existings, probed)

	log.Debugf("syncGuestnetworkSubIPs removes: %s add %s", jsonutils.Marshal(removes), jsonutils.Marshal(adds))

	if err := man.removeGuestnetworkSubIPs(ctx, userCred, guestnetwork, removes); err != nil {
		return err
	}
	if _, err := man.addGuestnetworkSubIPs(ctx, userCred, guestnetwork, api.GuestAddSubIpsInfo{
		SubIps:   adds,
		Reserved: true,
	}); err != nil {
		return err
	}
	return nil
}

func (man *SNetworkAddressManager) removeGuestnetworkSubIPs(ctx context.Context, userCred mcclient.TokenCredential, guestnetwork *SGuestnetwork, ipAddrs []string) error {
	q := man.queryByGuestnetworkId(guestnetwork.RowId)
	q = q.In("ip_addr", ipAddrs)

	var nas []SNetworkAddress
	if err := db.FetchModelObjects(man, q, &nas); err != nil {
		return errors.Wrapf(err, "removeGuestnetworkSubIPs by ipAddrs %s", strings.Join(ipAddrs, ", "))
	}

	for i := range nas {
		na := &nas[i]
		if err := db.DeleteModel(ctx, userCred, na); err != nil {
			return err
		}
	}
	return nil
}

func (man *SNetworkAddressManager) addGuestnetworkSubIPs(ctx context.Context, userCred mcclient.TokenCredential, guestnetwork *SGuestnetwork, input api.GuestAddSubIpsInfo) ([]string, error) {
	net, err := guestnetwork.GetNetwork()
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetwork")
	}

	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)

	var (
		usedAddrMap         = net.GetUsedAddresses(ctx)
		recentUsedAddrTable = GuestnetworkManager.getRecentlyReleasedIPAddresses(net.Id, net.getAllocTimoutDuration())
	)

	if input.Count == 0 {
		input.Count = len(input.SubIps)
	}
	if input.Count == 0 {
		// nil operation
		return nil, nil
	}

	addedIps := make([]string, 0)

	errs := make([]error, 0)
	for i := 0; i < input.Count; i++ {
		var candidate string
		if i < len(input.SubIps) {
			candidate = input.SubIps[i]
		}
		ipAddr, err := net.GetFreeIP(ctx, userCred, usedAddrMap, recentUsedAddrTable, candidate, input.AllocDir, input.Reserved, api.AddressTypeIPv4)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "GetFreeIP"))
			continue
		}

		m, err := db.NewModelObject(man)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "NewModelObject"))
			continue
		}
		na := m.(*SNetworkAddress)
		na.NetworkId = guestnetwork.NetworkId
		na.ParentType = api.NetworkAddressParentTypeGuestnetwork
		na.ParentId = strconv.FormatInt(guestnetwork.RowId, 10)
		na.Type = api.NetworkAddressTypeSubIP
		na.IpAddr = ipAddr
		if err := man.TableSpec().Insert(ctx, na); err != nil {
			errs = append(errs, errors.Wrapf(err, "addGuestnetworkSubIPs"))
			continue
		}
		usedAddrMap[ipAddr] = true
		addedIps = append(addedIps, ipAddr)
	}
	if len(errs) > 0 {
		return addedIps, errors.NewAggregate(errs)
	}
	return addedIps, nil
}

func (man *SNetworkAddressManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NetworkAddressCreateInput) (api.NetworkAddressCreateInput, error) {
	return input, errors.Wrap(httperrors.ErrNotSupported, "no supported")
}

func (na *SNetworkAddress) parentIdInt64() int64 {
	r, err := strconv.ParseInt(na.ParentId, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("networkaddress %s: invalid parent_id %s", na.Id, na.ParentId))
	}
	return r
}

func (na *SNetworkAddress) getGuestnetwork(ctx context.Context, userCred mcclient.TokenCredential) (*SGuestnetwork, error) {
	if na.ParentType == api.NetworkAddressParentTypeGuestnetwork {
		guestnetwork, err := GuestnetworkManager.fetchByRowId(ctx, userCred, na.parentIdInt64())
		if err != nil {
			return nil, err
		}
		return guestnetwork, nil
	}
	return nil, errors.Errorf("getGuestnetwork: expect parent_type %s, got %s", api.NetworkAddressParentTypeGuestnetwork, na.ParentType)
}

func (na *SNetworkAddress) getGuest(ctx context.Context, userCred mcclient.TokenCredential) (*SGuest, error) {
	guestnetwork, err := na.getGuestnetwork(ctx, userCred)
	if err != nil {
		return nil, err
	}
	guest := guestnetwork.GetGuest()
	if guest != nil {
		return guest, nil
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "getGuest: guestnetwork guest_id %s", guestnetwork.GuestId)
}

func (na *SNetworkAddress) getIVM(ctx context.Context, userCred mcclient.TokenCredential) (cloudprovider.ICloudVM, error) {
	guest, err := na.getGuest(ctx, userCred)
	if err != nil {
		return nil, err
	}
	if guest.ExternalId == "" {
		return nil, errors.Errorf("getIVM: guest %s(%s) has empty external_id", guest.Name, guest.Id)
	}
	ivm, err := guest.GetIVM(ctx)
	if err != nil {
		return nil, err
	}
	return ivm, nil
}

func (na *SNetworkAddress) getICloudNic(ctx context.Context, userCred mcclient.TokenCredential) (cloudprovider.ICloudNic, error) {
	guestnetwork, err := na.getGuestnetwork(ctx, userCred)
	if err != nil {
		return nil, err
	}

	iVM, err := na.getIVM(ctx, userCred)
	if err != nil {
		return nil, err
	}
	iNics, err := iVM.GetINics()
	if err != nil {
		return nil, err
	}
	for _, iNic := range iNics {
		if iNic.GetIP() == guestnetwork.IpAddr && iNic.GetMAC() == guestnetwork.MacAddr {
			return iNic, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "getICloudNic: no cloud nic with ip %s, mac %s", guestnetwork.IpAddr, guestnetwork.MacAddr)
}

/*func (na *SNetworkAddress) remoteAssignAddress(ctx context.Context, userCred mcclient.TokenCredential) error {
	if na.ParentType == api.NetworkAddressParentTypeGuestnetwork {
		guest, err := na.getGuest(ctx, userCred)
		if err != nil {
			return err
		}
		if guest.ExternalId != "" {
			iNic, err := na.getICloudNic(ctx, userCred)
			if err != nil {
				return err
			}
			if err := iNic.AssignAddress([]string{na.IpAddr}); err != nil {
				if errors.Cause(err) == cloudprovider.ErrAddressCountExceed {
					return httperrors.NewNotAcceptableError("exceed address count limit: %v", err)
				}
				return err
			}
		} else {
			NetworkAddressManager.submitGuestSyncTask(ctx, userCred, guest)
		}
	}
	return nil
}*/

func (na *SNetworkAddress) remoteUnassignAddress(ctx context.Context, userCred mcclient.TokenCredential) error {
	if na.ParentType == api.NetworkAddressParentTypeGuestnetwork {
		guest, err := na.getGuest(ctx, userCred)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound {
				return nil
			}
			return err
		}
		if guest.ExternalId != "" {
			iNic, err := na.getICloudNic(ctx, userCred)
			if err != nil {
				if errors.Cause(err) == errors.ErrNotFound {
					return nil
				}
				return err
			}
			if err := iNic.UnassignAddress([]string{na.IpAddr}); err != nil {
				return err
			}
		} else {
			NetworkAddressManager.submitGuestSyncTask(ctx, userCred, guest)
		}
	}
	return nil
}

func (na *SNetworkAddress) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := na.remoteUnassignAddress(ctx, userCred); err != nil {
		return err
	}
	return nil
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
		return nil, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ListItemFilter")
	}

	q, err = man.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, input.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(input.GuestId) > 0 {
		gnq := GuestnetworkManager.Query().SubQuery()
		gq := GuestManager.Query().
			In("id", input.GuestId).
			SubQuery()
		idq := man.Query("id").
			Equals("parent_type", api.NetworkAddressParentTypeGuestnetwork)
		idq = idq.Join(gnq, sqlchemy.Equals(gnq.Field("row_id"), idq.Field("parent_id")))
		idq = idq.Join(gq, sqlchemy.Equals(gq.Field("id"), gnq.Field("guest_id")))
		q = q.In("id", idq.SubQuery())
	}

	q, err = managedResourceFilterByAccount(
		ctx,
		q, input.ManagedResourceListInput, "network_id", func() *sqlchemy.SQuery {
			networks := NetworkManager.Query()
			wires := WireManager.Query("id", "vpc_id", "manager_id").SubQuery()
			vpcs := VpcManager.Query("id", "manager_id").SubQuery()
			networks = networks.Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
			networks = networks.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
			networks = networks.AppendField(networks.Field("id"))
			networks = networks.AppendField(sqlchemy.NewFunction(sqlchemy.NewCase().When(sqlchemy.IsNullOrEmpty(wires.Field("manager_id")), vpcs.Field("manager_id")).Else(wires.Field("manager_id")), "manager_id", false))
			subq := networks.SubQuery().Query()
			subq = subq.AppendField(subq.Field("id"))
			return subq
		})
	if err != nil {
		return nil, errors.Wrap(err, "ManagedResourceFilterByAccount")
	}

	return q, nil
}

func (man *SNetworkAddressManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkAddressListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, err
	}
	q, err = man.SStandaloneAnonResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
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

func (man *SNetworkAddressManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, manager db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = db.ApplyFilterByOwner(ctx, q, userCred, owner, scope,
		&man.SStandaloneAnonResourceBaseManager,
	)
	if owner != nil {
		var condVar, condVal string
		switch scope {
		case rbacscope.ScopeProject:
			condVar, condVal = "tenant_id", owner.GetProjectId()
		case rbacscope.ScopeDomain:
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

func (man *SNetworkAddressManager) submitGuestSyncTask(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) {
	man.delayedWorkManager.Submit(ctx, delayedwork.DelayedWorkRequest{
		ID:        guest.Id,
		SoftDelay: 2 * time.Second,
		HardDelay: 5 * time.Second,
		Func: func(ctx context.Context) {
			var (
				fwOnly       = false
				parentTaskId = ""
			)
			err := guest.StartSyncTask(ctx, userCred, fwOnly, parentTaskId)
			if err != nil {
				log.Errorf("guest StartSyncTask: %v", err)
			}
		},
	})
}

func (g *SGuest) PerformUpdateSubIps(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestUpdateSubIpsInput,
) (jsonutils.JSONObject, error) {
	gn, err := g.findGuestnetworkByInfo(input.ServerNetworkInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "getGuestnetworkByIpOrMac ip=%s mac=%s", input.IpAddr, input.Mac)
	}

	var iNic cloudprovider.ICloudNic
	if g.ExternalId != "" {
		// sync to cloud
		var err error
		iNic, err = g.getICloudNic(ctx, gn)
		if err != nil {
			return nil, errors.Wrap(err, "getICloudNic")
		}
	}

	errs := make([]error, 0)
	{
		err := NetworkAddressManager.removeGuestnetworkSubIPs(ctx, userCred, gn, input.RemoveSubIps)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "removeGuestnetworkSubIPs"))
		}
	}

	addedIps, err := NetworkAddressManager.addGuestnetworkSubIPs(ctx, userCred, gn, input.GuestAddSubIpsInfo)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "addGuestnetworkSubIPs"))
	}

	if g.ExternalId != "" {
		// sync to cloud
		if len(input.RemoveSubIps) > 0 {
			if err := iNic.UnassignAddress(addedIps); err != nil {
				errs = append(errs, errors.Wrapf(err, "UnassignAddress %s", addedIps))
			}
		}
		if len(addedIps) > 0 {
			if err := iNic.AssignAddress(addedIps); err != nil {
				if errors.Cause(err) == cloudprovider.ErrAddressCountExceed {
					errs = append(errs, httperrors.NewNotAcceptableError("exceed address count limit: %v", err))
				} else {
					errs = append(errs, errors.Wrapf(err, "AssignAddress %s", addedIps))
				}
			}
		}
	} else {
		NetworkAddressManager.submitGuestSyncTask(ctx, userCred, g)
	}

	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	return nil, nil
}

func (g *SGuest) PerformAddSubIps(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestAddSubIpsInput,
) (jsonutils.JSONObject, error) {
	gn, err := g.findGuestnetworkByInfo(input.ServerNetworkInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "getGuestnetworkByIpOrMac ip=%s mac=%s", input.IpAddr, input.Mac)
	}

	addedIps, err := NetworkAddressManager.addGuestnetworkSubIPs(ctx, userCred, gn, input.GuestAddSubIpsInfo)
	if err != nil {
		return nil, errors.Wrap(err, "addGuestnetworkSubIPs")
	}

	if g.ExternalId != "" && len(addedIps) > 0 {
		// sync to cloud
		iNic, err := g.getICloudNic(ctx, gn)
		if err != nil {
			return nil, errors.Wrap(err, "getICloudNic")
		}
		if err := iNic.AssignAddress(addedIps); err != nil {
			if errors.Cause(err) == cloudprovider.ErrAddressCountExceed {
				return nil, httperrors.NewNotAcceptableError("exceed address count limit: %v", err)
			}
			return nil, errors.Wrapf(err, "AssignAddress %s", addedIps)
		}
	} else {
		NetworkAddressManager.submitGuestSyncTask(ctx, userCred, g)
	}

	return nil, nil
}

func (g *SGuest) getICloudNic(ctx context.Context, gn *SGuestnetwork) (cloudprovider.ICloudNic, error) {
	ivm, err := g.GetIVM(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetIVM")
	}
	iNics, err := ivm.GetINics()
	if err != nil {
		return nil, errors.Wrap(err, "GetINics %s")
	}
	for _, iNic := range iNics {
		if iNic.GetIP() == gn.IpAddr && iNic.GetMAC() == gn.MacAddr {
			return iNic, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "no nic of ip %s mac %s", gn.IpAddr, gn.MacAddr)
}
