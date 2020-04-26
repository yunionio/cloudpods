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
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	ALL_NETWORK_TYPES = api.ALL_NETWORK_TYPES
)

type SNetworkManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SWireResourceBaseManager
}

var NetworkManager *SNetworkManager

func init() {
	NetworkManager = &SNetworkManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SNetwork{},
			"networks_tbl",
			"network",
			"networks",
		),
	}
	NetworkManager.SetVirtualObject(NetworkManager)
}

type SNetwork struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase
	SWireResourceBase

	IfnameHint string `width:"9" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 起始IP地址
	GuestIpStart string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	// 结束IP地址
	GuestIpEnd string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	// 掩码
	GuestIpMask int8 `nullable:"false" list:"user" update:"user" create:"required"`
	// 网关地址
	GuestGateway string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	// DNS
	GuestDns string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	// allow multiple dhcp, seperated by ","
	GuestDhcp string `width:"64" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	GuestDomain string `width:"128" charset:"ascii" nullable:"true" get:"user" update:"user"`

	GuestIp6Start string `width:"64" charset:"ascii" nullable:"true"`
	GuestIp6End   string `width:"64" charset:"ascii" nullable:"true"`
	GuestIp6Mask  int8   `nullable:"true"`
	GuestGateway6 string `width:"64" charset:"ascii" nullable:"true"`
	GuestDns6     string `width:"64" charset:"ascii" nullable:"true"`

	GuestDomain6 string `width:"128" charset:"ascii" nullable:"true"`

	VlanId int `nullable:"false" default:"1" list:"user" update:"user" create:"optional"`

	// 二层网络Id
	// WireId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// 服务器类型
	// example: server
	ServerType string `width:"16" charset:"ascii" default:"guest" nullable:"true" list:"user" update:"user" create:"optional"`

	// 分配策略
	AllocPolicy string `width:"16" charset:"ascii" nullable:"true" get:"user" update:"user" create:"optional"`

	AllocTimoutSeconds int `default:"0" nullable:"true" get:"admin"`
}

func (manager *SNetworkManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{WireManager},
	}
}

func (manager *SNetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SNetwork) GetNetworkInterfaces() ([]SNetworkInterface, error) {
	sq := NetworkinterfacenetworkManager.Query().SubQuery()
	q := NetworkInterfaceManager.Query()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("networkinterface_id"))).
		Filter(sqlchemy.Equals(sq.Field("network_id"), self.Id))
	networkinterfaces := []SNetworkInterface{}
	err := db.FetchModelObjects(NetworkInterfaceManager, q, &networkinterfaces)
	if err != nil {
		return nil, err
	}
	return networkinterfaces, nil
}

func (self *SNetwork) GetReservedIPs() ([]SReservedip, error) {
	reservedIps := []SReservedip{}
	q := ReservedipManager.Query().Equals("network_id", self.Id)
	err := db.FetchModelObjects(ReservedipManager, q, &reservedIps)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return reservedIps, nil
}

func (self *SNetwork) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetAllocatedNicCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetAllocatedNicCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("not an empty network")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SNetwork) GetTotalNicCount() (int, error) {
	total, err := self.GetAllocatedNicCount()
	if err != nil {
		return -1, err
	}
	cnt, err := self.GetReservedNicsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetNetworkInterfacesCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	return total, nil
}

func (self *SNetwork) GetAllocatedNicCount() (int, error) {
	total := 0
	cnt, err := self.GetGuestnicsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetGroupNicsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetBaremetalNicsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetLoadbalancerIpsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetDBInstanceIpsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetEipsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	return total, nil
}

/*验证elb network可用，并返回关联的region, zone,vpc, wire*/
func (self *SNetwork) ValidateElbNetwork(ipAddr net.IP) (*SCloudregion, *SZone, *SVpc, *SWire, error) {
	// 验证IP Address可用
	if ipAddr != nil {
		ipS := ipAddr.String()
		ip, err := netutils.NewIPV4Addr(ipS)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !self.IsAddressInRange(ip) {
			return nil, nil, nil, nil, httperrors.NewInputParameterError("address %s is not in the range of network %s(%s)",
				ipS, self.Name, self.Id)
		}

		used, err := self.isAddressUsed(ipS)
		if err != nil {
			return nil, nil, nil, nil, httperrors.NewInternalServerError("isAddressUsed fail %s", err)
		}
		if used {
			return nil, nil, nil, nil, httperrors.NewInputParameterError("address %s is already occupied", ipS)
		}
	}

	// 验证网络存在剩余地址空间
	freeCnt, err := self.getFreeAddressCount()
	if err != nil {
		return nil, nil, nil, nil, httperrors.NewInternalServerError("getFreeAddressCount fail %s", err)
	}
	if freeCnt <= 0 {
		return nil, nil, nil, nil, httperrors.NewNotAcceptableError("network %s(%s) has no free addresses",
			self.Name, self.Id)
	}

	// 验证网络可用
	wire := self.GetWire()
	if wire == nil {
		return nil, nil, nil, nil, fmt.Errorf("getting wire failed")
	}

	vpc := wire.GetVpc()
	if vpc == nil {
		return nil, nil, nil, nil, fmt.Errorf("getting vpc failed")
	}

	var zone *SZone
	if len(wire.ZoneId) > 0 {
		zone = wire.GetZone()
		if zone == nil {
			return nil, nil, nil, nil, fmt.Errorf("getting zone failed")
		}
	}

	region := wire.GetRegion()
	if region == nil {
		return nil, nil, nil, nil, fmt.Errorf("getting region failed")
	}

	return region, zone, vpc, wire, nil
}

func (self *SNetwork) GetGuestnicsCount() (int, error) {
	return GuestnetworkManager.Query().Equals("network_id", self.Id).IsFalse("virtual").CountWithError()
}

func (self *SNetwork) GetGroupNicsCount() (int, error) {
	return GroupnetworkManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetBaremetalNicsCount() (int, error) {
	return HostnetworkManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetReservedNicsCount() (int, error) {
	return ReservedipManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetLoadbalancerIpsCount() (int, error) {
	return LoadbalancernetworkManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetDBInstanceIpsCount() (int, error) {
	return DBInstanceNetworkManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetEipsCount() (int, error) {
	return ElasticipManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetNetworkInterfacesCount() (int, error) {
	sq := NetworkinterfacenetworkManager.Query("networkinterface_id").Equals("network_id", self.Id).Distinct().SubQuery()
	return NetworkInterfaceManager.Query().In("id", sq).CountWithError()
}

func (manager *SNetworkManager) NewClassicNetwork(wire *SWire) (*SNetwork, error) {
	_network, err := db.FetchByExternalId(manager, wire.Id)
	if err == nil {
		return _network.(*SNetwork), nil
	}
	if errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchByExternalId")
	}
	network := SNetwork{
		GuestIpStart: "0.0.0.0",
		GuestIpEnd:   "255.255.255.255",
		GuestIpMask:  0,
		GuestGateway: "0.0.0.0",
		ServerType:   api.NETWORK_TYPE_GUEST,
	}
	network.WireId = wire.Id
	network.SetModelManager(manager, &network)
	network.Name = fmt.Sprintf("emulate network for classic network with wire %s", wire.Id)
	network.ExternalId = wire.Id
	network.IsEmulated = true
	network.IsPublic = true
	network.PublicScope = "system"
	admin := auth.AdminCredential()
	network.DomainId = admin.GetProjectDomainId()
	network.ProjectId = admin.GetProjectId()
	network.Status = api.NETWORK_STATUS_UNAVAILABLE
	err = manager.TableSpec().Insert(&network)
	if err != nil {
		return nil, errors.Wrap(err, "Insert classic network")
	}
	return &network, nil
}

func (self *SNetwork) GetUsedAddresses() map[string]bool {
	used := make(map[string]bool)

	q := self.getUsedAddressQuery(true)
	results, err := q.AllStringMap()
	if err != nil {
		log.Errorf("GetUsedAddresses fail %s", err)
		return used
	}
	for _, result := range results {
		used[result["ip_addr"]] = true
	}
	return used
}

func (self *SNetwork) GetIPRange() netutils.IPV4AddrRange {
	return self.getIPRange()
}

func (self *SNetwork) getIPRange() netutils.IPV4AddrRange {
	start, _ := netutils.NewIPV4Addr(self.GuestIpStart)
	end, _ := netutils.NewIPV4Addr(self.GuestIpEnd)
	return netutils.NewIPV4AddrRange(start, end)
}

func isIpUsed(ipstr string, addrTable map[string]bool, recentUsedAddrTable map[string]bool) bool {
	_, ok := addrTable[ipstr]
	if !ok {
		recentUsed := false
		if recentUsedAddrTable != nil {
			if _, ok := recentUsedAddrTable[ipstr]; ok {
				recentUsed = true
			}
		}
		return recentUsed
	} else {
		return true
	}
}

func (self *SNetwork) getFreeIP(addrTable map[string]bool, recentUsedAddrTable map[string]bool, candidate string, allocDir api.IPAllocationDirection) (string, error) {
	iprange := self.getIPRange()
	// Try candidate first
	if len(candidate) > 0 {
		candIP, err := netutils.NewIPV4Addr(candidate)
		if err != nil {
			return "", err
		}
		if !iprange.Contains(candIP) {
			return "", httperrors.NewInputParameterError("candidate %s out of range", candidate)
		}
		if _, ok := addrTable[candidate]; !ok {
			return candidate, nil
		}
	}
	if len(self.AllocPolicy) > 0 && api.IPAllocationDirection(self.AllocPolicy) != api.IPAllocationNone {
		allocDir = api.IPAllocationDirection(self.AllocPolicy)
	}
	if len(allocDir) == 0 || allocDir == api.IPAllocationStepdown {
		ip, _ := netutils.NewIPV4Addr(self.GuestIpEnd)
		for iprange.Contains(ip) {
			if !isIpUsed(ip.String(), addrTable, recentUsedAddrTable) {
				return ip.String(), nil
			}
			ip = ip.StepDown()
		}
	} else {
		if allocDir == api.IPAllocationRadnom {
			iprange := self.getIPRange()
			const MAX_TRIES = 5
			for i := 0; i < MAX_TRIES; i += 1 {
				ip := iprange.Random()
				if !isIpUsed(ip.String(), addrTable, recentUsedAddrTable) {
					return ip.String(), nil
				}
			}
			// failed, fallback to IPAllocationStepup
		}
		ip, _ := netutils.NewIPV4Addr(self.GuestIpStart)
		for iprange.Contains(ip) {
			if !isIpUsed(ip.String(), addrTable, recentUsedAddrTable) {
				return ip.String(), nil
			}
			ip = ip.StepUp()
		}
	}
	return "", httperrors.NewInsufficientResourceError("Out of IP address")
}

func (self *SNetwork) GetFreeIP(ctx context.Context, userCred mcclient.TokenCredential, addrTable map[string]bool, recentUsedAddrTable map[string]bool, candidate string, allocDir api.IPAllocationDirection, reserved bool) (string, error) {
	// if reserved true, first try find IP in reserved IP pool
	if reserved {
		rip := ReservedipManager.GetReservedIP(self, candidate)
		if rip != nil {
			rip.Release(ctx, userCred, self)
			return candidate, nil
		}
		// return "", httperrors.NewInsufficientResourceError("Reserved address %s not found", candidate)
		// if not find, warning, then fallback to normal procedure
		log.Warningf("Reserved address %s not found", candidate)
	}
	if addrTable == nil {
		addrTable = self.GetUsedAddresses()
	}
	if recentUsedAddrTable == nil {
		recentUsedAddrTable = GuestnetworkManager.getRecentlyReleasedIPAddresses(self.Id, self.getAllocTimoutDuration())
	}
	cand, err := self.getFreeIP(addrTable, recentUsedAddrTable, candidate, allocDir)
	if err != nil {
		return "", err
	}
	return cand, nil
}

func (self *SNetwork) GetNetAddr() netutils.IPV4Addr {
	startIp, _ := netutils.NewIPV4Addr(self.GuestIpStart)
	return startIp.NetAddr(self.GuestIpMask)
}

func (self *SNetwork) GetDNS() string {
	if len(self.GuestDns) > 0 && len(self.GuestDomain) > 0 {
		return self.GuestDns
	} else {
		return options.Options.DNSServer
	}
}

func (self *SNetwork) GetDomain() string {
	if len(self.GuestDns) > 0 && len(self.GuestDomain) > 0 {
		return self.GuestDomain
	} else {
		return options.Options.DNSDomain
	}
}

func (self *SNetwork) GetRoutes() [][]string {
	ret := make([][]string, 0)
	routes := self.GetMetadataJson("static_routes", nil)
	if routes != nil {
		routesMap, err := routes.GetMap()
		if err != nil {
			return nil
		}
		for net, routeJson := range routesMap {
			route, _ := routeJson.GetString()
			ret = append(ret, []string{net, route})
		}
	}
	return ret
}

func (self *SNetwork) updateDnsRecord(nic *SGuestnetwork, isAdd bool) {
	guest := nic.GetGuest()
	self._updateDnsRecord(guest.Name, nic.IpAddr, isAdd)
}

func (self *SNetwork) _updateDnsRecord(name string, ipAddr string, isAdd bool) {
	if len(self.GuestDns) > 0 && len(self.GuestDomain) > 0 && len(ipAddr) > 0 {
		keyName := self.GetMetadata("dns_update_key_name", nil)
		keySecret := self.GetMetadata("dns_update_key_secret", nil)
		dnsSrv := self.GetMetadata("dns_update_server", nil)
		if len(dnsSrv) == 0 || !regutils.MatchIPAddr(dnsSrv) {
			dnsSrv = self.GuestDns
		}
		log.Infof("dns update %s %s isAdd=%t", ipAddr, dnsSrv, isAdd)
		if len(keyName) > 0 && len(keySecret) > 0 {
			/* netman.get_manager().dns_update(name,
			self.guest_domain, ip_addr, None,
			dns_srv, self.guest_dns6, key_name, key_secret,
			is_add) */
		}
		targets := self.getDnsUpdateTargets()
		if targets != nil {
			for srv, keys := range targets {
				for _, key := range keys {
					log.Debugf("Register %s %s", srv, key)
					/*
											netman.get_manager().dns_update(name,
						                            self.guest_domain, ip_addr, None,
						                            srv, None,
						                            key.get('key', None),
						                            key.get('secret', None),
						                            is_add)
					*/
				}
			}
		}
	}
}

func (self *SNetwork) updateGuestNetmap(nic *SGuestnetwork) {
	// TODO

}

func (self *SNetwork) UpdateBaremetalNetmap(nic *SHostnetwork, name string) {
	self.UpdateNetmap(nic.IpAddr, auth.AdminCredential().GetTenantId(), name)
}

func (self *SNetwork) UpdateNetmap(ip, project, name string) {
	// TODO ??
}

type DNSUpdateKeySecret struct {
	Key    string
	Secret string
}

func (self *SNetwork) getDnsUpdateTargets() map[string][]DNSUpdateKeySecret {
	targets := make(map[string][]DNSUpdateKeySecret)
	targetsJson := self.GetMetadataJson(api.EXTRA_DNS_UPDATE_TARGETS, nil)
	if targetsJson == nil {
		return nil
	} else {
		err := targetsJson.Unmarshal(&targets)
		if err != nil {
			return nil
		}
		return targets
	}
}

func (self *SNetwork) GetGuestIpv4StartAddress() netutils.IPV4Addr {
	addr, _ := netutils.NewIPV4Addr(self.GuestIpStart)
	return addr
}

func (self *SNetwork) IsExitNetwork() bool {
	return netutils.IsExitAddress(self.GetGuestIpv4StartAddress())
}

func (manager *SNetworkManager) getNetworksByWire(wire *SWire) ([]SNetwork, error) {
	return wire.getNetworks()
	/* nets := make([]SNetwork, 0)
	q := manager.Query().Equals("wire_id", wire.Id)
	err := db.FetchModelObjects(manager, q, &nets)
	if err != nil {
		log.Errorf("getNetworkByWire fail %s", err)
		return nil, err
	}
	return nets, nil */
}

func (manager *SNetworkManager) SyncNetworks(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire, nets []cloudprovider.ICloudNetwork, provider *SCloudprovider) ([]SNetwork, []cloudprovider.ICloudNetwork, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	localNets := make([]SNetwork, 0)
	remoteNets := make([]cloudprovider.ICloudNetwork, 0)
	syncResult := compare.SyncResult{}

	dbNets, err := manager.getNetworksByWire(wire)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := range dbNets {
		if taskman.TaskManager.IsInTask(&dbNets[i]) {
			syncResult.Error(fmt.Errorf("object in task"))
			return nil, nil, syncResult
		}
	}

	removed := make([]SNetwork, 0)
	commondb := make([]SNetwork, 0)
	commonext := make([]cloudprovider.ICloudNetwork, 0)
	added := make([]cloudprovider.ICloudNetwork, 0)

	err = compare.CompareSets(dbNets, nets, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudNetwork(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudNetwork(ctx, userCred, commonext[i], syncOwnerId, provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localNets = append(localNets, commondb[i])
			remoteNets = append(remoteNets, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudNetwork(ctx, userCred, added[i], wire, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localNets = append(localNets, *new)
			remoteNets = append(remoteNets, added[i])
			syncResult.Add()
		}
	}

	return localNets, remoteNets, syncResult
}

func (self *SNetwork) syncRemoveCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	if self.ExternalId == self.WireId {
		return nil
	}

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.NETWORK_STATUS_UNKNOWN, "Sync to remove")
	} else {
		err = self.RealDelete(ctx, userCred)

	}
	return err
}

func (self *SNetwork) SyncWithCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	vpc := self.GetVpc()
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		extNet.Refresh()
		self.Status = extNet.GetStatus()
		self.GuestIpStart = extNet.GetIpStart()
		self.GuestIpEnd = extNet.GetIpEnd()
		self.GuestIpMask = extNet.GetIpMask()
		self.GuestGateway = extNet.GetGateway()
		self.ServerType = extNet.GetServerType()

		self.AllocTimoutSeconds = extNet.GetAllocTimeoutSeconds()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudNetwork error %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, syncOwnerId, extNet, vpc.ManagerId)

	if provider != nil {
		self.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	return nil
}

func (manager *SNetworkManager) newFromCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork, wire *SWire, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SNetwork, error) {
	net := SNetwork{}
	net.SetModelManager(manager, &net)

	newName, err := db.GenerateName(manager, syncOwnerId, extNet.GetName())
	if err != nil {
		return nil, err
	}
	net.Name = newName
	net.Status = extNet.GetStatus()
	net.ExternalId = extNet.GetGlobalId()
	net.WireId = wire.Id
	net.GuestIpStart = extNet.GetIpStart()
	net.GuestIpEnd = extNet.GetIpEnd()
	net.GuestIpMask = extNet.GetIpMask()
	net.GuestGateway = extNet.GetGateway()
	net.ServerType = extNet.GetServerType()
	// net.IsPublic = extNet.GetIsPublic()
	// extScope := extNet.GetPublicScope()
	// if extScope == rbacutils.ScopeDomain && !consts.GetNonDefaultDomainProjects() {
	//	extScope = rbacutils.ScopeSystem
	// }
	// net.PublicScope = string(extScope)

	net.AllocTimoutSeconds = extNet.GetAllocTimeoutSeconds()

	err = manager.TableSpec().Insert(&net)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}

	vpc := wire.GetVpc()
	SyncCloudProject(userCred, &net, syncOwnerId, extNet, vpc.ManagerId)

	if provider != nil {
		net.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogEvent(&net, db.ACT_CREATE, net.GetShortDesc(ctx), userCred)

	return &net, nil
}

func (self *SNetwork) IsAddressInRange(address netutils.IPV4Addr) bool {
	return self.getIPRange().Contains(address)
}

func (self *SNetwork) isAddressUsed(address string) (bool, error) {
	q := self.getUsedAddressQuery(true)
	q = q.Equals("ip_addr", address)
	count, err := q.CountWithError()
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return false, errors.Wrap(err, "Query")
	}
	if count > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (manager *SNetworkManager) GetOnPremiseNetworkOfIP(ipAddr string, serverType string, isPublic tristate.TriState) (*SNetwork, error) {
	address, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return nil, err
	}
	q := manager.Query()
	wires := WireManager.Query().SubQuery()
	vpcs := VpcManager.Query().SubQuery()
	q = q.Join(wires, sqlchemy.Equals(q.Field("wire_id"), wires.Field("id")))
	q = q.Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))
	q = q.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
	if len(serverType) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("server_type"), serverType))
	}
	if isPublic.IsTrue() {
		q = q.Filter(sqlchemy.IsTrue(q.Field("is_public")))
	} else if isPublic.IsFalse() {
		q = q.Filter(sqlchemy.IsFalse(q.Field("is_public")))
	}

	nets := make([]SNetwork, 0)
	err = db.FetchModelObjects(manager, q, &nets)
	if err != nil {
		return nil, err
	}
	for _, n := range nets {
		if n.IsAddressInRange(address) {
			return &n, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (manager *SNetworkManager) allNetworksQ(providers []string, brands []string, cloudEnv string, rangeObjs []db.IStandaloneModel) *sqlchemy.SQuery {
	networks := manager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()
	q := networks.Query(networks.Field("id"))
	q = q.Join(hostwires, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
	q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL),
		sqlchemy.Equals(hosts.Field("host_status"), api.HOST_ONLINE)))
	return AttachUsageQuery(q, hosts, nil, nil, providers, brands, cloudEnv, rangeObjs)
}

func (manager *SNetworkManager) totalPortCountQ(
	scope rbacutils.TRbacScope,
	userCred mcclient.IIdentityProvider,
	providers []string,
	brands []string,
	cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) *sqlchemy.SQuery {
	q := manager.allNetworksQ(providers, brands, cloudEnv, rangeObjs)
	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", userCred.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", userCred.GetProjectId())
	}
	return manager.Query().In("id", q.Distinct().SubQuery())
}

type NetworkPortStat struct {
	Count    int
	CountExt int
}

func (manager *SNetworkManager) TotalPortCount(
	scope rbacutils.TRbacScope,
	userCred mcclient.IIdentityProvider,
	providers []string, brands []string, cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) NetworkPortStat {
	nets := make([]SNetwork, 0)
	err := manager.totalPortCountQ(
		scope,
		userCred,
		providers, brands, cloudEnv,
		rangeObjs,
	).All(&nets)
	if err != nil {
		log.Errorf("TotalPortCount: %v", err)
	}
	ct := 0
	ctExt := 0
	for _, net := range nets {
		count := net.getIPRange().AddressCount()
		if net.IsExitNetwork() {
			ctExt += count
		} else {
			ct += count
		}
	}
	return NetworkPortStat{Count: ct, CountExt: ctExt}
}

type SNicConfig struct {
	Mac    string
	Index  int8
	Ifname string
}

func parseNetworkInfo(userCred mcclient.TokenCredential, info *api.NetworkConfig) (*api.NetworkConfig, error) {
	if info.Network != "" {
		netObj, err := NetworkManager.FetchByIdOrName(userCred, info.Network)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(NetworkManager.Keyword(), info.Network)
			} else {
				return nil, err
			}
		}
		net := netObj.(*SNetwork)
		if net.ProjectId == userCred.GetProjectId() ||
			(db.IsDomainAllowGet(userCred, net) && net.DomainId == userCred.GetProjectDomainId()) ||
			db.IsAdminAllowGet(userCred, net) ||
			net.IsSharable(userCred) {
			info.Network = netObj.GetId()
		} else {
			return nil, httperrors.NewForbiddenError("no allow to access network %s", info.Network)
		}
	}

	if info.BwLimit == 0 {
		info.BwLimit = options.Options.DefaultBandwidth
	}
	return info, nil
}

func (self *SNetwork) GetFreeAddressCount() (int, error) {
	return self.getFreeAddressCount()
}

func (self *SNetwork) getFreeAddressCount() (int, error) {
	used, err := self.GetTotalNicCount()
	if err != nil {
		return -1, err
	}
	return self.getIPRange().AddressCount() - used, nil
}

func isValidNetworkInfo(userCred mcclient.TokenCredential, netConfig *api.NetworkConfig) error {
	if len(netConfig.Network) > 0 {
		netObj, err := NetworkManager.FetchByIdOrName(userCred, netConfig.Network)
		if err != nil {
			return httperrors.NewResourceNotFoundError("Network %s not found %s", err)
		}
		net := netObj.(*SNetwork)
		/*
			// scheduler do the check
			if !netConfig.Vip && !netConfig.Reserved && net.getFreeAddressCount() == 0 {
				return fmt.Errorf("Address exhausted in network %s")
			}*/
		if len(netConfig.Address) > 0 {
			ipAddr, err := netutils.NewIPV4Addr(netConfig.Address)
			if err != nil {
				return err
			}
			if !net.IsAddressInRange(ipAddr) {
				return httperrors.NewInputParameterError("Address %s not in range", netConfig.Address)
			}
			if netConfig.Reserved {
				// the privilege to access reserved ip
				if !db.IsAdminAllowList(userCred, ReservedipManager) {
					return httperrors.NewForbiddenError("Only system admin allowed to use reserved ip")
				}
				if ReservedipManager.GetReservedIP(net, netConfig.Address) == nil {
					return httperrors.NewInputParameterError("Address %s not reserved", netConfig.Address)
				}
			} else {
				used, err := net.isAddressUsed(netConfig.Address)
				if err != nil {
					return httperrors.NewInternalServerError("isAddressUsed fail %s", err)
				}
				if used {
					return httperrors.NewInputParameterError("Address %s has been used", netConfig.Address)
				}
			}
		}
		if netConfig.BwLimit > api.MAX_BANDWIDTH {
			return httperrors.NewInputParameterError("Bandwidth limit cannot exceed %dMbps", api.MAX_BANDWIDTH)
		}
		freeCnt, err := net.getFreeAddressCount()
		if err != nil {
			return httperrors.NewInternalServerError("getFreeAddressCount fail %s", err)
		}
		if freeCnt < 1 {
			return httperrors.NewInputParameterError("network %s(%s) has no free addresses", net.Name, net.Id)
		}
	}
	/* scheduler to the check
	else if ! netConfig.Vip {
		ct, ctExit := NetworkManager.to
	}
	*/
	return nil
}

func IsExitNetworkInfo(netConfig *api.NetworkConfig) bool {
	if len(netConfig.Network) > 0 {
		netObj, _ := NetworkManager.FetchById(netConfig.Network)
		net := netObj.(*SNetwork)
		if net.IsExitNetwork() {
			return true
		}
	} else if netConfig.Exit {
		return true
	}
	return false
}

func (self *SNetwork) GetPorts() int {
	return self.getIPRange().AddressCount()
}

func (self *SNetwork) getMoreDetails(ctx context.Context, out api.NetworkDetails, isList bool) (api.NetworkDetails, error) {
	out.Exit = false
	if self.IsExitNetwork() {
		out.Exit = true
	}
	out.Ports = self.GetPorts()
	out.PortsUsed, _ = self.GetTotalNicCount()

	out.Vnics, _ = self.GetGuestnicsCount()
	out.BmVnics, _ = self.GetBaremetalNicsCount()
	out.LbVnics, _ = self.GetLoadbalancerIpsCount()
	out.EipVnics, _ = self.GetEipsCount()
	out.GroupVnics, _ = self.GetGroupNicsCount()
	out.ReserveVnics, _ = self.GetReservedNicsCount()

	out.Routes = self.GetRoutes()
	out.Schedtags = GetSchedtagsDetailsToResourceV2(self, ctx)
	return out, nil
}

func (self *SNetwork) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.NetworkDetails, error) {
	return api.NetworkDetails{}, nil
}

func (manager *SNetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkDetails {
	rows := make([]api.NetworkDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	wireRows := manager.SWireResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.NetworkDetails{
			SharableVirtualResourceDetails: virtRows[i],
			WireResourceInfo:               wireRows[i],
		}
		rows[i], _ = objs[i].(*SNetwork).getMoreDetails(ctx, rows[i], isList)
	}

	return rows
}

func (self *SNetwork) AllowPerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "reserve-ip")
}

// 预留IP
// 预留的IP不会被调度使用
func (self *SNetwork) PerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkReserveIpInput) (jsonutils.JSONObject, error) {
	if len(input.Ips) == 0 {
		return nil, httperrors.NewMissingParameterError("ips")
	}

	var duration time.Duration
	if len(input.Duration) > 0 {
		bc, err := billing.ParseBillingCycle(input.Duration)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Duration %s invalid", input.Duration)
		}
		duration = bc.Duration()
	}

	for _, ip := range input.Ips {
		err := self.reserveIpWithDurationAndStatus(ctx, userCred, ip, input.Notes, duration, input.Status)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (self *SNetwork) reserveIpWithDuration(ctx context.Context, userCred mcclient.TokenCredential, ipstr string, notes string, duration time.Duration) error {
	return self.reserveIpWithDurationAndStatus(ctx, userCred, ipstr, notes, duration, "")
}

func (self *SNetwork) reserveIpWithDurationAndStatus(ctx context.Context, userCred mcclient.TokenCredential, ipstr string, notes string, duration time.Duration, status string) error {
	ipAddr, err := netutils.NewIPV4Addr(ipstr)
	if err != nil {
		return httperrors.NewInputParameterError("not a valid ip address %s: %s", ipstr, err)
	}
	if !self.IsAddressInRange(ipAddr) {
		return httperrors.NewInputParameterError("Address %s not in network", ipstr)
	}
	used, err := self.isAddressUsed(ipstr)
	if err != nil {
		return httperrors.NewInternalServerError("isAddressUsed fail %s", err)
	}
	if used {
		return httperrors.NewConflictError("Address %s has been used", ipstr)
	}
	err = ReservedipManager.ReserveIPWithDurationAndStatus(userCred, self, ipstr, notes, duration, status)
	if err != nil {
		return err
	}
	return nil
}

func (self *SNetwork) AllowPerformReleaseReservedIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "release-reserved-ip")
}

// 释放预留IP
func (self *SNetwork) PerformReleaseReservedIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkReleaseReservedIpInput) (jsonutils.JSONObject, error) {
	if len(input.Ip) == 0 {
		return nil, httperrors.NewMissingParameterError("ip")
	}
	rip := ReservedipManager.getReservedIP(self, input.Ip)
	if rip == nil {
		return nil, httperrors.NewInvalidStatusError("Address %s not reserved", input.Ip)
	}
	rip.Release(ctx, userCred, self)
	return nil, nil
}

func (self *SNetwork) AllowGetDetailsReservedIps(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "reserved-ips")
}

func (self *SNetwork) GetDetailsReservedIps(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	rips := ReservedipManager.GetReservedIPs(self)
	if rips == nil {
		return nil, httperrors.NewInternalServerError("get reserved ip error")
	}
	ripArray := jsonutils.NewArray()
	for i := 0; i < len(rips); i += 1 {
		ripArray.Add(jsonutils.NewString(rips[i].IpAddr))
	}
	ret := jsonutils.NewDict()
	ret.Add(ripArray, "reserved_ips")
	return ret, nil
}

func isValidMaskLen(maskLen int64) bool {
	if maskLen < 12 || maskLen > 30 {
		return false
	} else {
		return true
	}
}

func (self *SNetwork) ensureIfnameHint() {
	if self.IfnameHint != "" {
		return
	}
	hint, err := NetworkManager.newIfnameHint(self.Name)
	if err != nil {
		panic(errors.Wrap(err, "ensureIfnameHint: allocate hint"))
	}
	_, err = db.Update(self, func() error {
		self.IfnameHint = hint
		return nil
	})
	if err != nil {
		panic(errors.Wrap(err, "ensureIfnameHint: db update"))
	}
	log.Infof("network %s(%s): initialized ifname hint: %s", self.Name, self.Id, hint)
}

func (manager *SNetworkManager) newIfnameHint(hint string) (string, error) {
	isa := func(c byte) bool {
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	isn := func(c byte) bool {
		return (c >= '0' && c <= '9')
	}
	sani := func(r string) string {
		if r != "" && !isa(r[0]) {
			r = "a" + r
		}
		if len(r) > MAX_HINT_LEN {
			r = r[:MAX_HINT_LEN]
		}
		return r
	}
	rand := func(base string) (string, error) {
		if len(base) > HINT_BASE_LEN {
			base = base[:HINT_BASE_LEN]
		}
		for i := 0; i < 3; i++ {
			r := sani(base + rand.String(HINT_RAND_LEN))
			cnt, err := manager.Query().Equals("ifname_hint", r).CountWithError()
			if err == nil && cnt == 0 {
				return r, nil
			}
		}
		return "", fmt.Errorf("failed finding ifname hint after 3 tries")
	}

	r := ""
	for i := range hint {
		c := hint[i]
		if isa(c) || isn(c) || c == '_' {
			r += string(c)
		}
	}
	r = sani(r)

	if len(r) < 3 {
		return rand(r)
	}
	if cnt, err := manager.Query().Equals("ifname_hint", r).CountWithError(); err != nil {
		return "", err
	} else if cnt > 0 {
		return rand(r)
	}
	return r, nil
}

func (manager *SNetworkManager) validateEnsureWire(ctx context.Context, userCred mcclient.TokenCredential, input api.NetworkCreateInput) (w *SWire, v *SVpc, cr *SCloudregion, err error) {
	wObj, err := WireManager.FetchByIdOrName(userCred, input.Wire)
	if err != nil {
		err = errors.Wrapf(err, "wire %s", input.Wire)
		return
	}
	w = wObj.(*SWire)
	v = w.GetVpc()
	crObj, err := CloudregionManager.FetchById(v.CloudregionId)
	if err != nil {
		err = errors.Wrapf(err, "cloudregion %s", v.CloudregionId)
		return
	}
	cr = crObj.(*SCloudregion)
	return
}

func (manager *SNetworkManager) validateEnsureZoneVpc(ctx context.Context, userCred mcclient.TokenCredential, input api.NetworkCreateInput) (w *SWire, v *SVpc, cr *SCloudregion, err error) {
	zObj, err := ZoneManager.FetchByIdOrName(userCred, input.Zone)
	if err != nil {
		err = errors.Wrapf(err, "zone %s", input.Zone)
		return
	}
	z := zObj.(*SZone)

	vObj, err := VpcManager.FetchByIdOrName(userCred, input.Vpc)
	if err != nil {
		err = errors.Wrapf(err, "vpc %s", input.Vpc)
		return
	}
	v = vObj.(*SVpc)

	var wires []SWire
	// 华为云,ucloud wire zone_id 为空
	cr = z.GetRegion()
	if utils.IsInStringArray(cr.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
		wires, err = WireManager.getWiresByVpcAndZone(v, nil)
	} else {
		wires, err = WireManager.getWiresByVpcAndZone(v, z)
	}

	if err != nil {
		return
	} else if len(wires) > 1 {
		err = httperrors.NewConflictError("found %d wires for zone %s and vpc %s", len(wires), input.Zone, input.Vpc)
		return
	} else if len(wires) == 1 {
		w = &wires[0]
		return
	}
	// wire not found.  We auto create one for OneCloud vpc
	if cr.Provider == api.CLOUD_PROVIDER_ONECLOUD {
		w, err = v.initWire(ctx, z)
		if err != nil {
			err = errors.Wrapf(err, "vpc %s init wire", v.Id)
			return
		}
		return
	}
	err = httperrors.NewNotFoundError("wire not found for zone %s and vpc %s", input.Zone, input.Vpc)
	return
}

func (manager *SNetworkManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.NetworkCreateInput) (api.NetworkCreateInput, error) {
	if input.ServerType == "" {
		input.ServerType = api.NETWORK_TYPE_GUEST
	} else if !utils.IsInStringArray(input.ServerType, ALL_NETWORK_TYPES) {
		return input, httperrors.NewInputParameterError("Invalid server_type: %s", input.ServerType)
	}

	{
		if len(input.IfnameHint) == 0 {
			input.IfnameHint = input.Name
		}
		var err error
		input.IfnameHint, err = manager.newIfnameHint(input.IfnameHint)
		if err != nil {
			return input, httperrors.NewBadRequestError("cannot derive valid ifname hint: %v", err)
		}
	}

	var (
		ipRange netutils.IPV4AddrRange
	)
	if len(input.GuestIpPrefix) > 0 {
		prefix, err := netutils.NewIPV4Prefix(input.GuestIpPrefix)
		if err != nil {
			return input, httperrors.NewInputParameterError("ip_prefix error: %s", err)
		}
		ipRange = prefix.ToIPRange()
		input.GuestIpMask = int64(prefix.MaskLen)
		// 根据掩码得到合法的GuestIpPrefix
		input.GuestIpPrefix = prefix.String()
	} else {
		ipStart, err := netutils.NewIPV4Addr(input.GuestIpStart)
		if err != nil {
			return input, httperrors.NewInputParameterError("Invalid start ip: %s %s", input.GuestIpStart, err)
		}
		ipEnd, err := netutils.NewIPV4Addr(input.GuestIpEnd)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid end ip: %s %s", input.GuestIpEnd, err)
		}
		ipRange = netutils.NewIPV4AddrRange(ipStart, ipEnd)
	}

	if !isValidMaskLen(input.GuestIpMask) {
		return input, httperrors.NewInputParameterError("Invalid masklen %d", input.GuestIpMask)
	}

	if len(input.GuestDns) == 0 {
		input.GuestDns = options.Options.DNSServer
	}

	for key, ipStr := range map[string]string{"guest_gateway": input.GuestGateway, "guest_dns": input.GuestDns, "guest_dhcp": input.GuestDHCP} {
		if len(ipStr) > 0 {
			if key == "guest_dhcp" {
				ipList := strings.Split(ipStr, ",")
				for _, ipstr := range ipList {
					if !regutils.MatchIPAddr(ipstr) {
						return input, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipstr)
					}
				}
			} else if !regutils.MatchIPAddr(ipStr) {
				return input, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipStr)
			}
		}
	}

	var (
		wire   *SWire
		vpc    *SVpc
		region *SCloudregion
		err    error
	)
	if input.WireId != "" {
		input.Wire = input.WireId
	}
	if input.Wire != "" {
		wire, vpc, region, err = manager.validateEnsureWire(ctx, userCred, input)
		if err != nil {
			return input, err
		}
	} else if input.Zone != "" && input.Vpc != "" {
		wire, vpc, region, err = manager.validateEnsureZoneVpc(ctx, userCred, input)
		if err != nil {
			return input, err
		}
	} else {
		return input, httperrors.NewInputParameterError("zone and vpc info required when wire is absent")
	}
	input.WireId = wire.Id
	if vpc.Status != api.VPC_STATUS_AVAILABLE {
		return input, httperrors.NewInvalidStatusError("VPC not ready")
	}

	var (
		ipStart = ipRange.StartIp()
		ipEnd   = ipRange.EndIp()
	)
	if region.Provider == api.CLOUD_PROVIDER_ONECLOUD && vpc.Id != api.DEFAULT_VPC_ID {
		// reserve addresses for onecloud vpc networks
		masklen := int8(input.GuestIpMask)
		netAddr := ipStart.NetAddr(masklen)
		if masklen >= 30 {
			return input, httperrors.NewInputParameterError("subnet masklen should be smaller than 30")
		}
		if netAddr != ipEnd.NetAddr(masklen) {
			return input, httperrors.NewInputParameterError("start and end ip when masked are not in the same cidr subnet")
		}
		gateway := netAddr.StepUp()
		brdAddr := ipStart.BroadcastAddr(masklen)
		// NOTE
		//
		//  - reserve the 1st addr as gateway
		//  - reserve the last ip for broadcasting
		//  - reserve the 2nd-to-last for possible future use
		//
		// We do not allow split 192.168.1.0/24 into multiple ranges
		// like
		//
		//  - 192.168.1.50-192.168.1.100,
		//  - 192.168.1.100-192.168.1.200
		//
		//  This could complicate gateway setting and topology
		//  management without much benefit to end users
		ipStart = gateway.StepUp()
		ipEnd = brdAddr.StepDown().StepDown()
		input.GuestGateway = gateway.String()
	}

	{
		netRange := netutils.NewIPV4AddrRange(ipStart, ipEnd)
		if !vpc.containsIPV4Range(netRange) {
			return input, httperrors.NewInputParameterError("Network not in range of VPC cidrblock %s", vpc.CidrBlock)
		}
	}
	{
		nets := manager.getAllNetworks(wire.Id, "")
		if nets == nil {
			return input, httperrors.NewInternalServerError("query all networks fail")
		}
		if isOverlapNetworks(nets, ipStart, ipEnd) {
			return input, httperrors.NewInputParameterError("Conflict address space with existing networks")
		}
	}

	input.GuestIpStart = ipStart.String()
	input.GuestIpEnd = ipEnd.String()
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SNetwork) validateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.NetworkUpdateInput) (api.NetworkUpdateInput, error) {
	var startIp, endIp netutils.IPV4Addr
	var err error

	ipStartStr := input.GuestIpStart
	ipEndStr := input.GuestIpEnd

	if len(ipStartStr) > 0 || len(ipEndStr) > 0 {
		if len(ipStartStr) > 0 {
			startIp, err = netutils.NewIPV4Addr(ipStartStr)
			if err != nil {
				return input, httperrors.NewInputParameterError("Invalid start ip: %s %s", ipStartStr, err)
			}
		} else {
			startIp, _ = netutils.NewIPV4Addr(self.GuestIpStart)
		}
		if len(ipEndStr) > 0 {
			endIp, err = netutils.NewIPV4Addr(ipEndStr)
			if err != nil {
				return input, httperrors.NewInputParameterError("invalid end ip: %s %s", ipEndStr, err)
			}
		} else {
			endIp, _ = netutils.NewIPV4Addr(self.GuestIpEnd)
		}

		if startIp > endIp {
			tmp := startIp
			startIp = endIp
			endIp = tmp
		}

		nets := NetworkManager.getAllNetworks(self.WireId, self.Id)
		if nets == nil {
			return input, httperrors.NewInternalServerError("query all networks fail")
		}

		if isOverlapNetworks(nets, startIp, endIp) {
			return input, httperrors.NewInputParameterError("Conflict address space with existing networks")
		}

		netRange := netutils.NewIPV4AddrRange(startIp, endIp)
		vpc := self.GetVpc()
		if !vpc.containsIPV4Range(netRange) {
			return input, httperrors.NewInputParameterError("Network not in range of VPC cidrblock %s", vpc.CidrBlock)
		}

		usedMap := self.GetUsedAddresses()
		for usedIpStr := range usedMap {
			usedIp, _ := netutils.NewIPV4Addr(usedIpStr)
			if !netRange.Contains(usedIp) {
				return input, httperrors.NewInputParameterError("Address been assigned out of new range")
			}
		}

		input.GuestIpStart = startIp.String()
		input.GuestIpEnd = endIp.String()
	}

	if input.GuestIpMask != nil {
		maskLen64 := int64(*input.GuestIpMask)
		if !isValidMaskLen(maskLen64) {
			return input, httperrors.NewInputParameterError("Invalid masklen %d", maskLen64)
		}
	}

	for key, ipStr := range map[string]string{
		"guest_gateway": input.GuestGateway,
		"guest_dns":     input.GuestDns,
		"guest_dhcp":    input.GuestDhcp,
	} {
		if len(ipStr) > 0 {
			if key == "guest_dhcp" {
				ipList := strings.Split(ipStr, ",")
				for _, ipstr := range ipList {
					if !regutils.MatchIPAddr(ipstr) {
						return input, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipstr)
					}
				}
			} else if !regutils.MatchIPAddr(ipStr) {
				return input, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipStr)
			}

		}
	}
	return input, nil
}

func (self *SNetwork) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.NetworkUpdateInput) (api.NetworkUpdateInput, error) {
	if !self.isManaged() && !self.isOneCloudVpcNetwork() {
		var err error
		input, err = self.validateUpdateData(ctx, userCred, query, input)
		if err != nil {
			return input, errors.Wrap(err, "validateUpdateData")
		}
	} else {
		input.GuestIpStart = ""
		input.GuestIpEnd = ""
		input.GuestIpMask = nil
		input.GuestGateway = ""
		input.GuestDns = ""
		input.GuestDomain = ""
		input.GuestDhcp = ""
	}

	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = self.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SNetworkManager) getAllNetworks(wireId, excludeId string) []SNetwork {
	nets := make([]SNetwork, 0)
	q := manager.Query().Equals("wire_id", wireId)
	if len(excludeId) > 0 {
		q = q.NotEquals("id", excludeId)
	}
	err := db.FetchModelObjects(manager, q, &nets)
	if err != nil {
		log.Errorf("getAllNetworks fail %s", err)
		return nil
	}
	return nets
}

func isOverlapNetworks(nets []SNetwork, startIp netutils.IPV4Addr, endIp netutils.IPV4Addr) bool {
	ipRange := netutils.NewIPV4AddrRange(startIp, endIp)
	for i := 0; i < len(nets); i += 1 {
		ipRange2 := nets[i].getIPRange()
		if ipRange2.IsOverlap(ipRange) {
			return true
		}
	}
	return false
}

func (self *SNetwork) IsManaged() bool {
	wire := self.GetWire()
	if wire == nil {
		return false
	}
	return wire.IsManaged()
}

func (self *SNetwork) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("public_scope") {
		if self.ServerType == api.NETWORK_TYPE_GUEST && !self.IsManaged() {
			if db.IsAdminAllowPerform(userCred, self, "public") && ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				self.SetShare(rbacutils.ScopeSystem)
			} else if db.IsDomainAllowPerform(userCred, self, "public") && ownerId.GetProjectId() == userCred.GetProjectId() {
				self.SetShare(rbacutils.ScopeDomain)
			} else {
				self.SetShare(rbacutils.ScopeNone)
			}
		} else {
			self.SetShare(rbacutils.ScopeNone)
		}
		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(self.PublicScope))
	}
	return self.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SNetwork) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	vpc := self.GetVpc()
	if vpc != nil && vpc.IsManaged() {
		task, err := taskman.TaskManager.NewTask(ctx, "NetworkCreateTask", self, userCred, nil, "", "", nil)
		if err != nil {
			log.Errorf("networkcreateTask create fail: %s", err)
		} else {
			task.ScheduleRun(nil)
		}
	} else {
		self.SetStatus(userCred, api.NETWORK_STATUS_AVAILABLE, "")
		self.ClearSchedDescCache()
	}
}

func (self *SNetwork) GetPrefix() (netutils.IPV4Prefix, error) {
	addr, err := netutils.NewIPV4Addr(self.GuestIpStart)
	if err != nil {
		return netutils.IPV4Prefix{}, err
	}
	addr = addr.NetAddr(self.GuestIpMask)
	return netutils.IPV4Prefix{Address: addr, MaskLen: self.GuestIpMask}, nil
}

func (self *SNetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SNetwork delete do nothing")
	self.SetStatus(userCred, api.NETWORK_STATUS_START_DELETE, "")
	return nil
}

func (self *SNetwork) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.StartDeleteNetworkTask(ctx, userCred)
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SNetwork) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	self.SetStatus(userCred, api.NETWORK_STATUS_DELETED, "real delete")
	self.ClearSchedDescCache()
	networkinterfaces, err := self.GetNetworkInterfaces()
	if err != nil {
		return errors.Wrap(err, "GetNetworkInterfaces")
	}
	for i := range networkinterfaces {
		err = networkinterfaces[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "networkinterface.purge %s(%s)", networkinterfaces[i].Name, networkinterfaces[i].Id)
		}
	}
	reservedIps, err := self.GetReservedIPs()
	if err != nil {
		return errors.Wrap(err, "GetReservedNicsCount")
	}
	for i := range reservedIps {
		err = reservedIps[i].Release(ctx, userCred, self)
		if err != nil {
			return errors.Wrapf(err, "reservedIps.Release %s(%d)", reservedIps[i].IpAddr, reservedIps[i].Id)
		}
	}
	return self.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SNetwork) StartDeleteNetworkTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "NetworkDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start NetworkDeleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SNetwork) GetINetwork() (cloudprovider.ICloudNetwork, error) {
	wire := self.GetWire()
	if wire == nil {
		msg := "No wire for this network????"
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	iwire, err := wire.GetIWire()
	if err != nil {
		return nil, err
	}
	return iwire.GetINetworkById(self.GetExternalId())
}

func (self *SNetwork) isManaged() bool {
	if len(self.ExternalId) > 0 {
		return true
	} else {
		return false
	}
}

func (self *SNetwork) isOneCloudVpcNetwork() bool {
	vpc := self.GetVpc()
	region := self.GetRegion()
	if region.Provider == api.CLOUD_PROVIDER_ONECLOUD && vpc.Id != api.DEFAULT_VPC_ID {
		return true
	}
	return false
}

func parseIpToIntArray(ip string) ([]int, error) {
	ipSp := strings.Split(strings.Trim(ip, "."), ".")
	if len(ipSp) > 4 {
		return nil, httperrors.NewInputParameterError("Parse Ip Failed")
	}
	ipIa := []int{}
	for i := 0; i < len(ipSp); i++ {
		val, err := strconv.Atoi(ipSp[i])
		if err != nil {
			return nil, httperrors.NewInputParameterError("Parse Ip Failed")
		}
		if val < 0 || val > 255 {
			return nil, httperrors.NewInputParameterError("Parse Ip Failed")
		}
		ipIa = append(ipIa, val)
	}
	return ipIa, nil
}

// IP子网列表
func (manager *SNetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.NetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SWireResourceBaseManager.ListItemFilter(ctx, q, userCred, input.WireFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemFilter")
	}

	if input.Usable != nil && *input.Usable {
		wires := WireManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()
		providerSQ := cloudproviders.Query(cloudproviders.Field("id")).Filter(
			sqlchemy.AND(
				sqlchemy.IsTrue(cloudproviders.Field("enabled")),
				sqlchemy.In(cloudproviders.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS),
				sqlchemy.In(cloudproviders.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS),
			),
		)
		regions := CloudregionManager.Query().SubQuery()

		sq := wires.Query(wires.Field("id")).
			Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
			Join(zones, sqlchemy.OR(sqlchemy.Equals(wires.Field("zone_id"), zones.Field("id")), sqlchemy.IsNullOrEmpty(wires.Field("zone_id")))).
			Join(regions, sqlchemy.Equals(zones.Field("cloudregion_id"), regions.Field("id"))).
			Filter(sqlchemy.AND(
				sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE),
				sqlchemy.Equals(zones.Field("status"), api.ZONE_ENABLE),
				sqlchemy.Equals(regions.Field("status"), api.CLOUD_REGION_STATUS_INSERVER),
				sqlchemy.OR(
					sqlchemy.In(vpcs.Field("manager_id"), providerSQ.SubQuery()),
					sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")),
				),
			))
		q = q.In("wire_id", sq.SubQuery()).Equals("status", api.NETWORK_STATUS_AVAILABLE)
	}

	hostStr := input.Host
	if len(hostStr) > 0 {
		hostObj, err := HostManager.FetchByIdOrName(userCred, hostStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), hostStr)
		}
		host := hostObj.(*SHost)
		sq := HostwireManager.Query("wire_id").Equals("host_id", hostObj.GetId())
		if len(host.OvnVersion) > 0 {
			wireQuery := WireManager.Query("id").IsNotNull("vpc_id")
			q = q.Filter(sqlchemy.OR(
				sqlchemy.In(q.Field("wire_id"), wireQuery.SubQuery()),
				sqlchemy.In(q.Field("wire_id"), sq.SubQuery())),
			)
		} else {
			q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
		}
	}

	if len(input.Ip) > 0 {
		ipIa, err := parseIpToIntArray(input.Ip)
		if err != nil {
			return nil, err
		}

		ipSa := []string{"0", "0", "0", "0"}
		for i := range ipIa {
			ipSa[i] = strconv.Itoa(ipIa[i])
		}
		fullIp := strings.Join(ipSa, ".")

		ipField := sqlchemy.INET_ATON(sqlchemy.NewStringField(fullIp))
		ipStart := sqlchemy.INET_ATON(q.Field("guest_ip_start"))
		ipEnd := sqlchemy.INET_ATON(q.Field("guest_ip_end"))

		ipCondtion := sqlchemy.OR(sqlchemy.Between(ipField, ipStart, ipEnd), sqlchemy.Contains(q.Field("guest_ip_start"), input.Ip), sqlchemy.Contains(q.Field("guest_ip_end"), input.Ip))
		q = q.Filter(ipCondtion)
	}

	if len(input.IfnameHint) > 0 {
		q = q.In("ifname_hint", input.IfnameHint)
	}
	if len(input.GuestIpStart) > 0 {
		q = q.In("guest_ip_start", input.GuestIpStart)
	}
	if len(input.GuestIpEnd) > 0 {
		q = q.In("guest_ip_end", input.GuestIpEnd)
	}
	if len(input.GuestIpMask) > 0 {
		q = q.In("guest_ip_mask", input.GuestIpMask)
	}
	if len(input.GuestGateway) > 0 {
		q = q.In("guest_gateway", input.GuestGateway)
	}
	if len(input.GuestDns) > 0 {
		q = q.In("guest_dns", input.GuestDns)
	}
	if len(input.GuestDhcp) > 0 {
		q = q.In("guest_dhcp", input.GuestDhcp)
	}
	if len(input.GuestDomain) > 0 {
		q = q.In("guest_domain", input.GuestDomain)
	}
	if len(input.GuestIp6Start) > 0 {
		q = q.In("guest_ip6_start", input.GuestIp6Start)
	}
	if len(input.GuestIp6End) > 0 {
		q = q.In("guest_ip6_end", input.GuestIp6End)
	}
	if len(input.GuestIp6Mask) > 0 {
		q = q.In("guest_ip6_mask", input.GuestIp6Mask)
	}
	if len(input.GuestGateway6) > 0 {
		q = q.In("guest_gateway6", input.GuestGateway6)
	}
	if len(input.GuestDns6) > 0 {
		q = q.In("guest_dns6", input.GuestDns6)
	}
	if len(input.GuestDomain6) > 0 {
		q = q.In("guest_domain6", input.GuestDomain6)
	}
	if len(input.VlanId) > 0 {
		q = q.In("vlan_id", input.VlanId)
	}
	if len(input.ServerType) > 0 {
		q = q.In("server_type", input.ServerType)
	}
	if len(input.AllocPolicy) > 0 {
		q = q.In("alloc_policy", input.AllocPolicy)
	}

	return q, nil
}

func (manager *SNetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.NetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SWireResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.WireFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SNetworkManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SWireResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SNetworkManager) InitializeData() error {
	// set network status
	networks := make([]SNetwork, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &networks)
	if err != nil {
		return err
	}
	for _, n := range networks {
		if n.ExternalId != "" {
			var statusNew string
			if n.WireId != "" && n.Status == api.NETWORK_STATUS_INIT {
				statusNew = api.NETWORK_STATUS_AVAILABLE
			}
			db.Update(&n, func() error {
				if statusNew != "" {
					n.Status = statusNew
				}
				return nil
			})
		} else {
			var ifnameHintNew string
			if n.IfnameHint == "" {
				ifnameHintNew = n.Name
			}
			db.Update(&n, func() error {
				if ifnameHintNew != "" {
					n.IfnameHint = ifnameHintNew
				}
				return nil
			})
		}
	}
	return nil
}

func (self *SNetwork) ValidateUpdateCondition(ctx context.Context) error {
	/*if len(self.ExternalId) > 0 {
		return httperrors.NewConflictError("Cannot update external resource")
	}*/
	return self.SSharableVirtualResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SNetwork) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

// 清除IP子网数据
// 要求IP子网内没有被分配IP,若清除接入云,要求接入云账号处于禁用状态
func (self *SNetwork) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkPurgeInput) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	vpc := self.GetVpc()
	if vpc != nil && len(vpc.ExternalId) > 0 {
		provider := vpc.GetCloudprovider()
		if provider != nil && provider.GetEnabled() {
			return nil, httperrors.NewInvalidStatusError("Cannot purge network on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (self *SNetwork) AllowPerformSplit(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "split")
}

func (self *SNetwork) AllowPerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "merge")
}

func (manager *SNetworkManager) handleNetworkIdChange(ctx context.Context, args *networkIdChangeArgs) error {
	var handlers = []networkIdChangeHandler{
		GuestnetworkManager,
		HostnetworkManager,
		ReservedipManager,
		GroupnetworkManager,
		LoadbalancernetworkManager,
		LoadbalancerManager,
	}

	errs := []error{}
	for _, h := range handlers {
		if err := h.handleNetworkIdChange(ctx, args); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		err := errors.NewAggregate(errs)
		return httperrors.NewGeneralError(err)
	}
	return nil
}

// 合并IP子网
// 将两个相连的IP子网合并成一个IP子网
func (self *SNetwork) PerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkMergeInput) (jsonutils.JSONObject, error) {
	if len(input.Target) == 0 {
		return nil, httperrors.NewMissingParameterError("target")
	}
	iNet, err := NetworkManager.FetchByIdOrName(userCred, input.Target)
	if err == sql.ErrNoRows {
		err = httperrors.NewNotFoundError("Network %s not found", input.Target)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, err.Error(), userCred, false)
		return nil, err
	} else if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, err.Error(), userCred, false)
		return nil, err
	}

	net := iNet.(*SNetwork)
	if net == nil {
		err = fmt.Errorf("Network is nil")
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, err.Error(), userCred, false)
		return nil, err
	}
	if self.WireId != net.WireId || self.GuestGateway != net.GuestGateway {
		err = httperrors.NewInputParameterError("Invalid Target Network: %s", input.Target)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, err.Error(), userCred, false)
		return nil, err
	}

	var startIp, endIp string
	ipNE, _ := netutils.NewIPV4Addr(net.GuestIpEnd)
	ipNS, _ := netutils.NewIPV4Addr(net.GuestIpStart)
	ipSS, _ := netutils.NewIPV4Addr(self.GuestIpStart)
	ipSE, _ := netutils.NewIPV4Addr(self.GuestIpEnd)

	if ipNE.StepUp() == ipSS {
		startIp, endIp = net.GuestIpStart, self.GuestIpEnd
	} else if ipSE.StepUp() == ipNS {
		startIp, endIp = self.GuestIpStart, net.GuestIpEnd
	} else {
		note := "Incontinuity Network for %s and %s"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE,
			fmt.Sprintf(note, self.Name, net.Name), userCred, false)
		return nil, httperrors.NewBadRequestError(note, self.Name, net.Name)
	}

	lockman.LockClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))
	defer lockman.ReleaseClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))

	_, err = db.Update(net, func() error {
		net.GuestIpStart = startIp
		net.GuestIpEnd = endIp
		return nil
	})
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, err.Error(), userCred, false)
		return nil, err
	}

	if err := NetworkManager.handleNetworkIdChange(ctx, &networkIdChangeArgs{
		action:   logclient.ACT_MERGE,
		oldNet:   self,
		newNet:   net,
		userCred: userCred,
	}); err != nil {
		return nil, err
	}

	note := map[string]string{"start_ip": startIp, "end_ip": endIp}
	db.OpsLog.LogEvent(self, db.ACT_MERGE, note, userCred)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_MERGE, note, userCred, true)

	if err = self.RealDelete(ctx, userCred); err != nil {
		return nil, err
	}
	note = map[string]string{"network": self.Id}
	db.OpsLog.LogEvent(self, db.ACT_DELETE, note, userCred)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_DELOCATE, note, userCred, true)
	return nil, nil
}

// 分割IP子网
// 将一个IP子网分割成两个子网,仅本地IDC支持此操作
func (self *SNetwork) PerformSplit(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkSplitInput) (jsonutils.JSONObject, error) {
	if len(self.ExternalId) > 0 {
		return nil, httperrors.NewNotSupportedError("only on premise support this operation")
	}

	if len(input.SplitIp) == 0 {
		return nil, httperrors.NewMissingParameterError("split_ip")
	}

	if !regutils.MatchIPAddr(input.SplitIp) {
		return nil, httperrors.NewInputParameterError("Invalid IP %s", input.SplitIp)
	}
	if input.SplitIp == self.GuestIpStart {
		return nil, httperrors.NewInputParameterError("Split IP %s is the start ip", input.SplitIp)
	}

	iSplitIp, err := netutils.NewIPV4Addr(input.SplitIp)
	if err != nil {
		return nil, err
	}
	if !self.IsAddressInRange(iSplitIp) {
		return nil, httperrors.NewInputParameterError("Split IP %s out of range", input.SplitIp)
	}

	lockman.LockClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))
	defer lockman.ReleaseClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))

	if len(input.Name) > 0 {
		if err := db.NewNameValidator(NetworkManager, userCred, input.Name, ""); err != nil {
			return nil, httperrors.NewInputParameterError("Duplicate name %s", input.Name)
		}
	} else {
		input.Name, err = db.GenerateName(NetworkManager, userCred, fmt.Sprintf("%s#", self.Name))
		if err != nil {
			return nil, httperrors.NewInternalServerError("GenerateName fail %s", err)
		}
	}

	network := &SNetwork{}
	network.Name = input.Name
	network.IfnameHint, err = NetworkManager.newIfnameHint(input.Name)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Generate ifname hint failed %s", err)
	}
	network.GuestIpStart = input.SplitIp
	network.GuestIpEnd = self.GuestIpEnd
	network.GuestIpMask = self.GuestIpMask
	network.GuestGateway = self.GuestGateway
	network.GuestDns = self.GuestDns
	network.GuestDhcp = self.GuestDhcp
	network.GuestDomain = self.GuestDomain
	network.VlanId = self.VlanId
	network.WireId = self.WireId
	network.ServerType = self.ServerType
	network.IsPublic = self.IsPublic
	network.Status = self.Status
	network.ProjectId = self.ProjectId
	network.DomainId = self.DomainId
	// network.UserId = self.UserId
	network.IsSystem = self.IsSystem
	network.Description = self.Description

	err = NetworkManager.TableSpec().Insert(network)
	if err != nil {
		return nil, err
	}
	network.SetModelManager(NetworkManager, network)

	db.Update(self, func() error {
		self.GuestIpEnd = iSplitIp.StepDown().String()
		return nil
	})

	if err := NetworkManager.handleNetworkIdChange(ctx, &networkIdChangeArgs{
		action:   logclient.ACT_SPLIT,
		oldNet:   self,
		newNet:   network,
		userCred: userCred,
	}); err != nil {
		return nil, err
	}

	note := map[string]string{"split_ip": input.SplitIp, "end_ip": network.GuestIpEnd}
	db.OpsLog.LogEvent(self, db.ACT_SPLIT, note, userCred)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_SPLIT, note, userCred, true)
	db.OpsLog.LogEvent(network, db.ACT_CREATE, map[string]string{"network": self.Id}, userCred)
	return nil, nil
}

func (manager *SNetworkManager) AllowPerformTryCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "try-create-network")
}

func (manager *SNetworkManager) PerformTryCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkTryCreateNetworkInput) (jsonutils.JSONObject, error) {
	if len(input.Ip) == 0 {
		return nil, httperrors.NewMissingParameterError("ip")
	}
	ipV4, err := netutils.NewIPV4Addr(input.Ip)
	if err != nil {
		return nil, httperrors.NewInputParameterError("ip")
	}
	if input.Mask == 0 {
		return nil, httperrors.NewMissingParameterError("mask")
	}
	if len(input.ServerType) == 0 {
		return nil, httperrors.NewMissingParameterError("server_type")
	}
	if input.ServerType != api.NETWORK_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Only support server type %s", api.NETWORK_TYPE_BAREMETAL)
	}
	if !input.IsOnPremise {
		return nil, httperrors.NewBadRequestError("Only support on premise network")
	}

	var (
		ipV4NetAddr = ipV4.NetAddr(int8(input.Mask))
		nm          *SNetwork
		matched     bool
	)

	q := NetworkManager.Query().Equals("server_type", input.ServerType).Equals("guest_ip_mask", input.Mask)

	listQuery := api.NetworkListInput{}
	err = query.Unmarshal(&listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "query.Unmarshal")
	}
	q, err = managedResourceFilterByAccount(q, listQuery.ManagedResourceListInput, "wire_id", func() *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		subq := wires.Query(wires.Field("id"))
		subq = subq.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		return subq
	})
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByAccount")
	}

	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := db.NewModelObject(NetworkManager)
		if err != nil {
			return nil, err
		}
		err = q.Row2Struct(rows, item)
		if err != nil {
			return nil, err
		}
		n := item.(*SNetwork)
		if n.GetIPRange().Contains(ipV4) {
			nm = n
			matched = true
			break
		} else if nIpV4, _ := netutils.NewIPV4Addr(n.GuestIpStart); nIpV4.NetAddr(n.GuestIpMask) == ipV4NetAddr {
			nm = n
			matched = false
			break
		}
	}

	ret := jsonutils.NewDict()
	if nm == nil {
		ret.Set("find_matched", jsonutils.JSONFalse)
		return ret, nil
	}
	ret.Set("find_matched", jsonutils.JSONTrue)
	ret.Set("wire_id", jsonutils.NewString(nm.WireId))
	if !matched {
		log.Infof("Find same subnet network %s %s/%d", nm.Name, nm.GuestGateway, nm.GuestIpMask)
		newNetwork := new(SNetwork)
		newNetwork.SetModelManager(NetworkManager, newNetwork)
		newNetwork.GuestIpStart = input.Ip
		newNetwork.GuestIpEnd = input.Ip
		newNetwork.GuestGateway = nm.GuestGateway
		newNetwork.GuestIpMask = int8(input.Mask)
		newNetwork.GuestDns = nm.GuestDns
		newNetwork.GuestDhcp = nm.GuestDhcp
		newNetwork.WireId = nm.WireId
		newNetwork.ServerType = input.ServerType
		newNetwork.IsPublic = nm.IsPublic
		newNetwork.ProjectId = userCred.GetProjectId()
		newNetwork.DomainId = userCred.GetProjectDomainId()
		newName, err := db.GenerateName(NetworkManager, userCred, fmt.Sprintf("%s#", nm.Name))
		if err != nil {
			return nil, httperrors.NewInternalServerError("GenerateName fail %s", err)
		}
		newNetwork.Name = newName

		err = NetworkManager.TableSpec().Insert(newNetwork)
		if err != nil {
			return nil, err
		}
		err = newNetwork.CustomizeCreate(ctx, userCred, userCred, query, input.JSON(input))
		if err != nil {
			return nil, err
		}
		newNetwork.PostCreate(ctx, userCred, userCred, query, input.JSON(input))
	}
	return ret, nil
}

func (network *SNetwork) getAllocTimoutDuration() time.Duration {
	tos := network.AllocTimoutSeconds
	if tos < options.Options.MinimalIpAddrReusedIntervalSeconds {
		tos = options.Options.MinimalIpAddrReusedIntervalSeconds
	}
	return time.Duration(tos) * time.Second
}

func (network *SNetwork) GetSchedtags() []SSchedtag {
	return GetSchedtags(NetworkschedtagManager, network.Id)
}

func (network *SNetwork) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(network).(*jsonutils.JSONDict)
}

func (network *SNetwork) AllowPerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return AllowPerformSetResourceSchedtag(network, ctx, userCred, query, data)
}

func (network *SNetwork) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(network, ctx, userCred, query, data)
}

func (network *SNetwork) GetSchedtagJointManager() ISchedtagJointManager {
	return NetworkschedtagManager
}

func (network *SNetwork) ClearSchedDescCache() error {
	wire := network.GetWire()
	if wire == nil {
		return nil
	}
	return wire.clearHostSchedDescCache()
}

func (network *SNetwork) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	ret, err := network.SSharableVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	network.ClearSchedDescCache()
	return ret, nil
}

func (network *SNetwork) AllowGetDetailsAddresses(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return network.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, network, "addresses")
}

func (network *SNetwork) getUsedAddressQuery(addrOnly bool) *sqlchemy.SQuery {
	guestnetworks := GuestnetworkManager.Query().Equals("network_id", network.Id).SubQuery()
	var guestNetQ *sqlchemy.SQuery
	if addrOnly {
		guestNetQ = guestnetworks.Query(
			guestnetworks.Field("ip_addr"),
		)
	} else {
		guests := GuestManager.Query().SubQuery()
		guestNetQ = guestnetworks.Query(
			guestnetworks.Field("ip_addr"),
			guestnetworks.Field("mac_addr"),
			sqlchemy.NewStringField(GuestManager.KeywordPlural()).Label("owner_type"),
			guestnetworks.Field("guest_id").Label("owner_id"),
			guests.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			guestnetworks.Field("created_at"),
		).Join(
			guests,
			sqlchemy.Equals(
				guests.Field("id"),
				guestnetworks.Field("guest_id"),
			),
		)
	}

	groupnetworks := GroupnetworkManager.Query().Equals("network_id", network.Id).SubQuery()
	var groupNetQ *sqlchemy.SQuery
	if addrOnly {
		groupNetQ = groupnetworks.Query(
			groupnetworks.Field("ip_addr"),
		)
	} else {
		groups := GroupManager.Query().SubQuery()
		groupNetQ = groupnetworks.Query(
			groupnetworks.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(GroupManager.KeywordPlural()).Label("owner_type"),
			groupnetworks.Field("group_id").Label("owner_id"),
			groups.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			groupnetworks.Field("created_at"),
		).Join(
			groups,
			sqlchemy.Equals(
				groups.Field("id"),
				groupnetworks.Field("group_id"),
			),
		)
	}

	hostnetworks := HostnetworkManager.Query().Equals("network_id", network.Id).SubQuery()
	var hostNetQ *sqlchemy.SQuery
	if addrOnly {
		hostNetQ = hostnetworks.Query(
			hostnetworks.Field("ip_addr"),
		)
	} else {
		hosts := HostManager.Query().SubQuery()
		hostNetQ = hostnetworks.Query(
			hostnetworks.Field("ip_addr"),
			hostnetworks.Field("mac_addr"),
			sqlchemy.NewStringField(HostManager.KeywordPlural()).Label("owner_type"),
			hostnetworks.Field("baremetal_id").Label("owner_id"),
			hosts.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			hostnetworks.Field("created_at"),
		).Join(
			hosts,
			sqlchemy.Equals(
				hosts.Field("id"),
				hostnetworks.Field("baremetal_id"),
			),
		)
	}

	reserved := ReservedipManager.Query().Equals("network_id", network.Id).SubQuery()
	var reservedQ *sqlchemy.SQuery
	if addrOnly {
		reservedQ = reserved.Query(
			reserved.Field("ip_addr"),
		)
	} else {
		reservedQ = reserved.Query(
			reserved.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ReservedipManager.KeywordPlural()).Label("owner_type"),
			reserved.Field("id").Label("owner_id"),
			reserved.Field("notes").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			reserved.Field("created_at"),
		)
	}
	reservedQ = reservedQ.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(reserved.Field("expired_at")),
		sqlchemy.GT(reserved.Field("expired_at"), time.Now()),
	))

	lbnetworks := LoadbalancernetworkManager.Query().Equals("network_id", network.Id).SubQuery()
	var lbNetQ *sqlchemy.SQuery
	if addrOnly {
		lbNetQ = lbnetworks.Query(
			lbnetworks.Field("ip_addr"),
		)
	} else {
		loadbalancers := LoadbalancerManager.Query().SubQuery()
		lbNetQ = lbnetworks.Query(
			lbnetworks.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(LoadbalancerManager.KeywordPlural()).Label("owner_type"),
			lbnetworks.Field("loadbalancer_id").Label("owner_id"),
			loadbalancers.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			lbnetworks.Field("created_at"),
		).Join(
			loadbalancers,
			sqlchemy.Equals(
				loadbalancers.Field("id"),
				lbnetworks.Field("loadbalancer_id"),
			),
		)
	}

	elasticips := ElasticipManager.Query().Equals("network_id", network.Id).SubQuery()
	var eipQ *sqlchemy.SQuery
	if addrOnly {
		eipQ = elasticips.Query(
			elasticips.Field("ip_addr"),
		)
	} else {
		eipQ = elasticips.Query(
			elasticips.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ElasticipManager.KeywordPlural()).Label("owner_type"),
			elasticips.Field("id").Label("owner_id"),
			elasticips.Field("name").Label("owner"),
			elasticips.Field("associate_id"),
			elasticips.Field("associate_type"),
			elasticips.Field("created_at"),
		)
	}

	netifnetworks := NetworkinterfacenetworkManager.Query().Equals("network_id", network.Id).SubQuery()
	var netifsQ *sqlchemy.SQuery
	if addrOnly {
		netifsQ = netifnetworks.Query(
			netifnetworks.Field("ip_addr"),
		)
	} else {
		netifs := NetworkInterfaceManager.Query().SubQuery()
		netifsQ = netifnetworks.Query(
			netifnetworks.Field("ip_addr"),
			netifs.Field("mac").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkInterfaceManager.KeywordPlural()).Label("owner_type"),
			netifnetworks.Field("networkinterface_id").Label("owner_id"),
			netifs.Field("name").Label("owner"),
			netifs.Field("associate_id"),
			netifs.Field("associate_type"),
			netifnetworks.Field("created_at"),
		).Join(
			netifs,
			sqlchemy.Equals(
				netifnetworks.Field("networkinterface_id"),
				netifs.Field("id"),
			),
		)
	}

	return sqlchemy.Union(guestNetQ, groupNetQ, hostNetQ, reservedQ, lbNetQ, eipQ, netifsQ).Query()
}

type SNetworkAddressList []api.SNetworkAddress

func (a SNetworkAddressList) Len() int      { return len(a) }
func (a SNetworkAddressList) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SNetworkAddressList) Less(i, j int) bool {
	ipI, _ := netutils.NewIPV4Addr(a[i].IpAddr)
	ipJ, _ := netutils.NewIPV4Addr(a[j].IpAddr)
	return ipI < ipJ
}

func (network *SNetwork) GetDetailsAddresses(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	netAddrs := make([]api.SNetworkAddress, 0)
	q := network.getUsedAddressQuery(false)
	err := q.All(&netAddrs)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	sort.Sort(SNetworkAddressList(netAddrs))

	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(netAddrs), "addresses")
	return result, nil
}

func (net *SNetwork) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return net.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, net, "syncstatus")
}

// 同步接入云IP子网状态
// 本地IDC不支持此操作
func (net *SNetwork) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkSyncInput) (jsonutils.JSONObject, error) {
	return net.PerformSync(ctx, userCred, query, input)
}

func (net *SNetwork) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return net.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, net, "sync")
}

// 同步接入云IP子网状态
// 本地IDC不支持此操作
func (net *SNetwork) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NetworkSyncInput) (jsonutils.JSONObject, error) {
	vpc := net.GetVpc()
	if vpc != nil && vpc.IsManaged() {
		return nil, StartResourceSyncStatusTask(ctx, userCred, net, "NetworkSyncstatusTask", "")
	}
	return nil, httperrors.NewUnsupportOperationError("on-premise network cannot sync status")
}

func (net *SNetwork) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) bool {
	return net.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, net, "status")
}

// 更改IP子网状态
func (net *SNetwork) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	if len(input.Status) == 0 {
		return nil, httperrors.NewMissingParameterError("status")
	}
	vpc := net.GetVpc()
	if vpc != nil && vpc.IsManaged() {
		return nil, httperrors.NewUnsupportOperationError("managed network cannot change status")
	}
	if !utils.IsInStringArray(input.Status, []string{api.NETWORK_STATUS_AVAILABLE, api.NETWORK_STATUS_UNAVAILABLE}) {
		return nil, httperrors.NewInputParameterError("invalid status %s", input.Status)
	}
	return net.SSharableVirtualResourceBase.PerformStatus(ctx, userCred, query, input)
}

func (net *SNetwork) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{
		net.SSharableVirtualResourceBase.GetChangeOwnerCandidateDomainIds(),
	}
	wire := net.GetWire()
	if wire != nil {
		candidates = append(candidates, db.ISharableChangeOwnerCandidateDomainIds(wire))
	}
	return db.ISharableMergeChangeOwnerCandidateDomainIds(net, candidates...)
}

func (manager *SNetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SWireResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SWireResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
