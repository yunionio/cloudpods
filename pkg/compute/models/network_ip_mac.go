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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNetworkIpMacManager struct {
	db.SStandaloneAnonResourceBaseManager
}

var NetworkIpMacManager *SNetworkIpMacManager

func GetNetworkIpMacManager() *SNetworkIpMacManager {
	if NetworkIpMacManager != nil {
		return NetworkIpMacManager
	}
	NetworkIpMacManager = &SNetworkIpMacManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SNetworkIpMac{},
			"network_ip_macs_tbl",
			"network_ip_mac",
			"network_ip_macs",
		),
	}
	NetworkIpMacManager.SetVirtualObject(NetworkIpMacManager)
	return NetworkIpMacManager
}

func init() {
	GetNetworkIpMacManager()
}

type SNetworkIpMac struct {
	db.SStandaloneAnonResourceBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" index:"true" create:"required"`
	// MAC地址
	MacAddr string `width:"32" charset:"ascii" nullable:"false" list:"user" index:"true" update:"user" create:"required"`
	// IPv4地址
	IpAddr string `width:"16" charset:"ascii" nullable:"false" list:"user" index:"true" update:"user" create:"required"`
}

func (manager *SNetworkIpMacManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SNetworkIpMac) GetOwnerId() mcclient.IIdentityProvider {
	iNetwork, _ := NetworkManager.FetchById(self.NetworkId)
	if iNetwork != nil {
		return iNetwork.GetOwnerId()
	}
	return nil
}

func (manager *SNetworkIpMacManager) NetworkMacAddrInUse(networkId, macAddr string) (bool, error) {
	count, err := manager.Query().
		Equals("network_id", networkId).
		Equals("mac_addr", macAddr).CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (manager *SNetworkIpMacManager) NetworkIpAddrInUse(networkId, ipAddr string) (bool, error) {
	count, err := manager.Query().
		Equals("network_id", networkId).
		Equals("ip_addr", ipAddr).CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (manager *SNetworkIpMacManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NetworkIpMacCreateInput,
) (api.NetworkIpMacCreateInput, error) {
	if input.NetworkId == "" {
		return input, httperrors.NewMissingParameterError("network_id")
	}
	if input.IpAddr == "" {
		return input, httperrors.NewMissingParameterError("ip_addr")
	}
	if input.MacAddr == "" {
		return input, httperrors.NewMissingParameterError("mac_addr")
	}

	iNetwork, err := NetworkManager.FetchByIdOrName(userCred, input.NetworkId)
	if err == sql.ErrNoRows {
		return input, httperrors.NewNotFoundError("network %s not found", input.NetworkId)
	} else if err != nil {
		return input, errors.Wrap(err, "fetch network")
	}
	network := iNetwork.(*SNetwork)
	if network.ServerType != api.NETWORK_TYPE_GUEST {
		return input, httperrors.NewBadRequestError("network type %s can't set ip mac", network.ServerType)
	}

	input.NetworkId = network.Id
	input.MacAddr = strings.ToLower(input.MacAddr)
	return input, manager.validateIpMac(input.IpAddr, input.MacAddr, network)
}

func (self *SNetworkIpMac) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.NetworkIpMacUpdateInput,
) (api.NetworkIpMacUpdateInput, error) {
	if input.IpAddr == "" && input.MacAddr == "" {
		return input, httperrors.NewInputParameterError("missing update field")
	}

	if input.IpAddr != "" && input.IpAddr != self.IpAddr {
		iNetwork, err := NetworkManager.FetchByIdOrName(userCred, self.NetworkId)
		if err != nil {
			return input, errors.Wrap(err, "fetch network")
		}
		network := iNetwork.(*SNetwork)
		if !network.Contains(input.IpAddr) {
			return input, errors.Errorf("network %s not contains ip addr %s", network.GetName(), input.IpAddr)
		}
		if ipInUse, err := NetworkIpMacManager.NetworkIpAddrInUse(self.NetworkId, input.IpAddr); err != nil {
			return input, errors.Wrap(err, "check ip addr in use")
		} else if ipInUse {
			return input, httperrors.NewBadRequestError("ip addr %s is in use", input.IpAddr)
		}
	} else {
		input.IpAddr = self.IpAddr
	}

	input.MacAddr = strings.ToLower(input.MacAddr)
	if input.MacAddr != "" && input.MacAddr != self.MacAddr {
		if !utils.IsMatchMacAddr(input.MacAddr) {
			return input, errors.Errorf("mac address %s is not valid", input.MacAddr)
		}
		if macInUse, err := NetworkIpMacManager.NetworkMacAddrInUse(self.NetworkId, input.MacAddr); err != nil {
			return input, errors.Wrap(err, "check mac addr in use")
		} else if macInUse {
			return input, httperrors.NewBadRequestError("mac addr %s is in use", input.MacAddr)
		}
	} else {
		input.MacAddr = self.MacAddr
	}

	if gn, err := GuestnetworkManager.getGuestNicByIP(self.NetworkId, input.IpAddr); err != nil {
		return input, errors.Wrap(err, "failed get guest nic")
	} else if gn != nil && gn.MacAddr != input.MacAddr {
		return input, errors.Errorf("input ip mac conflict with guest %s nic %d", gn.GuestId, gn.Index)
	}

	return input, nil
}

func (manager *SNetworkIpMacManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.NetworkIpMacListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ListItemFilter")
	}
	if input.NetworkId != "" {
		iNetwork, err := NetworkManager.FetchByIdOrName(userCred, input.NetworkId)
		if err != nil {
			return q, errors.Wrap(err, "fetch network")
		}
		q = q.Equals("network_id", iNetwork.GetId())
	}
	if input.MacAddr != nil {
		q = q.In("mac_addr", input.MacAddr)
	}
	if input.IpAddr != nil {
		q = q.In("ip_addr", input.IpAddr)
	}
	return q, nil
}

func (manager *SNetworkIpMacManager) GetMacFromIp(networkId, ipAddr string) string {
	q := manager.Query("mac_addr").Equals("network_id", networkId).Equals("ip_addr", ipAddr)
	if q.Count() != 1 {
		return ""
	}
	var nim = new(SNetworkIpMac)
	if err := q.First(nim); err != nil {
		log.Errorf("failed get mac addr form ip %s: %s", ipAddr, err)
		return ""
	}
	return nim.MacAddr
}

func (manager *SNetworkIpMacManager) validateIpMac(ip, mac string, network *SNetwork) error {
	if !network.Contains(ip) {
		return httperrors.NewBadRequestError("network %s not contains ip addr %s", network.GetName(), ip)
	}
	if ipInUse, err := manager.NetworkIpAddrInUse(network.Id, ip); err != nil {
		return errors.Wrap(err, "check ip addr in use")
	} else if ipInUse {
		return httperrors.NewBadRequestError("ip addr %s is in use", ip)
	}

	if !utils.IsMatchMacAddr(mac) {
		return httperrors.NewBadRequestError("mac address %s is not valid", mac)
	}
	if macInUse, err := manager.NetworkMacAddrInUse(network.Id, mac); err != nil {
		return errors.Wrap(err, "check mac addr in use")
	} else if macInUse {
		return httperrors.NewBadRequestError("mac addr %s is in use", mac)
	}

	if gn, err := GuestnetworkManager.getGuestNicByIP(network.Id, ip); err != nil {
		return errors.Wrap(err, "failed get guest nic")
	} else if gn != nil && gn.MacAddr != mac {
		return httperrors.NewBadRequestError("input ip mac conflict with guest %s nic %d", gn.GuestId, gn.Index)
	}
	return nil
}

func (self *SNetworkIpMac) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	iNetwork, _ := NetworkManager.FetchByIdOrName(userCred, self.NetworkId)
	note := fmt.Sprintf("create ip %s mac %s bind", self.IpAddr, self.MacAddr)
	db.OpsLog.LogEvent(iNetwork, db.ACT_IP_MAC_BIND, note, userCred)
	logclient.AddActionLogWithContext(ctx, iNetwork, logclient.ACT_IP_MAC_BIND, note, userCred, true)
}

func (self *SNetworkIpMac) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) {
	iNetwork, _ := NetworkManager.FetchByIdOrName(userCred, self.NetworkId)
	note := fmt.Sprintf("update ip %s mac %s bind", self.IpAddr, self.MacAddr)
	db.OpsLog.LogEvent(iNetwork, db.ACT_IP_MAC_BIND, note, userCred)
	logclient.AddActionLogWithContext(ctx, iNetwork, logclient.ACT_IP_MAC_BIND, note, userCred, true)
}

func (self *SNetworkIpMac) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	iNetwork, _ := NetworkManager.FetchByIdOrName(userCred, self.NetworkId)
	note := fmt.Sprintf("delete ip %s mac %s bind", self.IpAddr, self.MacAddr)
	db.OpsLog.LogEvent(iNetwork, db.ACT_IP_MAC_BIND, note, userCred)
	logclient.AddActionLogWithContext(ctx, iNetwork, logclient.ACT_IP_MAC_BIND, note, userCred, true)
}

func (manager *SNetworkIpMacManager) PerformBatchCreate(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkIpMacBatchCreateInput,
) (jsonutils.JSONObject, error) {
	iNetwork, err := NetworkManager.FetchByIdOrName(userCred, input.NetworkId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch network")
	}
	network := iNetwork.(*SNetwork)
	if network.ServerType != api.NETWORK_TYPE_GUEST {
		return nil, errors.Errorf("network type %s can't set ip mac", network.ServerType)
	}
	input.NetworkId = network.Id

	var errs = []error{}
	var insertedIpMacs = map[string]string{}
	for ip, mac := range input.IpMac {
		if err := manager.validateIpMac(ip, mac, network); err != nil {
			errs = append(errs, err)
			continue
		}
		nim := &SNetworkIpMac{
			NetworkId: input.NetworkId,
			IpAddr:    ip,
			MacAddr:   mac,
		}
		if err := manager.TableSpec().Insert(ctx, nim); err != nil {
			errs = append(errs, err)
			continue
		}
		insertedIpMacs[ip] = mac
	}
	note := fmt.Sprintf("create ip macs bind %v", insertedIpMacs)
	db.OpsLog.LogEvent(iNetwork, db.ACT_IP_MAC_BIND, note, userCred)
	logclient.AddActionLogWithContext(ctx, iNetwork, logclient.ACT_IP_MAC_BIND, note, userCred, true)

	if len(errs) == 0 {
		return nil, nil
	}
	return nil, errors.NewAggregate(errs)
}
func (manager *SNetworkIpMacManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkIpMacDetails {
	rows := make([]api.NetworkIpMacDetails, len(objs))

	baseRows := manager.SStandaloneAnonResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.NetworkIpMacDetails{
			StandaloneAnonResourceDetails: baseRows[i],
		}
	}
	return rows
}
