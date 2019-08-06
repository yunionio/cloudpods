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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	ALL_NETWORK_TYPES = api.ALL_NETWORK_TYPES
)

type IPAddlocationDirection string

const (
	IPAllocationStepdown IPAddlocationDirection = "stepdown"
	IPAllocationStepup   IPAddlocationDirection = "stepup"
	IPAllocationRadnom   IPAddlocationDirection = "random"
	IPAllocationNone     IPAddlocationDirection = "none"
	IPAllocationDefault                         = ""
)

type SNetworkManager struct {
	db.SSharableVirtualResourceBaseManager
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
	NetworkManager.NameLength = 9
	NetworkManager.NameRequireAscii = true
	NetworkManager.SetVirtualObject(NetworkManager)
}

type SNetwork struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	GuestIpStart string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	GuestIpEnd   string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	GuestIpMask  int8   `nullable:"false" list:"user" update:"user" create:"required"`                            // Column(TINYINT, nullable=False)
	GuestGateway string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`  // Column(VARCHAR(16, charset='ascii'), nullable=True)
	GuestDns     string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`  // Column(VARCHAR(16, charset='ascii'), nullable=True)
	GuestDhcp    string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`  // Column(VARCHAR(16, charset='ascii'), nullable=True)

	GuestDomain string `width:"128" charset:"ascii" nullable:"true" get:"user" update:"user"` // Column(VARCHAR(128, charset='ascii'), nullable=True)

	GuestIp6Start string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	GuestIp6End   string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	GuestIp6Mask  int8   `nullable:"true"`                            // Column(TINYINT, nullable=True)
	GuestGateway6 string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	GuestDns6     string `width:"64" charset:"ascii" nullable:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=True)

	GuestDomain6 string `width:"128" charset:"ascii" nullable:"true"` // Column(VARCHAR(128, charset='ascii'), nullable=True)

	VlanId int `nullable:"false" default:"1" list:"user" update:"user" create:"optional"` // Column(Integer, nullable=False, default=1)

	WireId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	// IsChanged = Column(Boolean, nullable=False, default=False)

	ServerType string `width:"16" charset:"ascii" default:"guest" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AllocPolicy string `width:"16" charset:"ascii" nullable:"true" get:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AllocTimoutSeconds int `default:"0" nullable:"true" get:"admin"`
}

func (manager *SNetworkManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{WireManager},
	}
}

func (self *SNetwork) GetWire() *SWire {
	w, _ := WireManager.FetchById(self.WireId)
	if w != nil {
		return w.(*SWire)
	}
	return nil
}

func (self *SNetwork) GetVpc() *SVpc {
	wire := self.GetWire()
	if wire != nil {
		return wire.getVpc()
	}
	return nil
}

func (manager *SNetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SNetwork) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetTotalNicCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetTotalNicCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("not an empty network")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SNetwork) GetTotalNicCount() (int, error) {
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
	cnt, err = self.GetReservedNicsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetLoadbalancerIpsCount()
	if err != nil {
		return -1, err
	}
	total += cnt
	cnt, err = self.GetEipsCount()
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

	vpc := wire.getVpc()
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

	region := wire.getRegion()
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

func (self *SNetwork) GetEipsCount() (int, error) {
	return ElasticipManager.Query().Equals("network_id", self.Id).CountWithError()
}

func (self *SNetwork) GetNetworkInterfacesCount() (int, error) {
	sq := NetworkinterfacenetworkManager.Query("networkinterface_id").Equals("network_id", self.Id).Distinct().SubQuery()
	return NetworkInterfaceManager.Query().In("id", sq).CountWithError()
}

func (self *SNetwork) GetUsedAddresses() map[string]bool {
	used := make(map[string]bool)

	for _, tbl := range []*sqlchemy.SSubQuery{
		GuestnetworkManager.Query().SubQuery(),
		GroupnetworkManager.Query().SubQuery(),
		HostnetworkManager.Query().SubQuery(),
		ReservedipManager.Query().SubQuery(),
		LoadbalancernetworkManager.Query().SubQuery(),
		ElasticipManager.Query().SubQuery(),
		NetworkinterfacenetworkManager.Query().SubQuery(),
	} {
		q := tbl.Query(tbl.Field("ip_addr")).Equals("network_id", self.Id)
		rows, err := q.Rows()
		if err != nil {
			log.Errorf("GetUsedAddresses query fail: %s", err)
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var ip string
			err = rows.Scan(&ip)
			if err != nil {
				log.Errorf("GetUsedAddresses scan fail: %s", err)
				return nil
			}
			used[ip] = true
		}
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

func (self *SNetwork) getFreeIP(addrTable map[string]bool, recentUsedAddrTable map[string]bool, candidate string, allocDir IPAddlocationDirection) (string, error) {
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
	if len(self.AllocPolicy) > 0 && IPAddlocationDirection(self.AllocPolicy) != IPAllocationNone {
		allocDir = IPAddlocationDirection(self.AllocPolicy)
	}
	if len(allocDir) == 0 || allocDir == IPAllocationStepdown {
		ip, _ := netutils.NewIPV4Addr(self.GuestIpEnd)
		for iprange.Contains(ip) {
			if !isIpUsed(ip.String(), addrTable, recentUsedAddrTable) {
				return ip.String(), nil
			}
			ip = ip.StepDown()
		}
	} else {
		if allocDir == IPAllocationRadnom {
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

func (self *SNetwork) GetFreeIP(ctx context.Context, userCred mcclient.TokenCredential, addrTable map[string]bool, recentUsedAddrTable map[string]bool, candidate string, allocDir IPAddlocationDirection, reserved bool) (string, error) {
	if reserved {
		rip := ReservedipManager.GetReservedIP(self, candidate)
		if rip == nil {
			return "", httperrors.NewInsufficientResourceError("Reserved address %s not found", candidate)
		}
		rip.Release(ctx, userCred, self)
		return candidate, nil
	} else {
		cand, err := self.getFreeIP(addrTable, recentUsedAddrTable, candidate, allocDir)
		if err != nil {
			return "", err
		}
		return cand, nil
	}
}

func (self *SNetwork) GetUsedIfnames() map[string]bool {
	used := make(map[string]bool)
	tbl := GuestnetworkManager.Query().SubQuery()
	q := tbl.Query(tbl.Field("ifname")).Equals("network_id", self.Id)
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("GetUsedIfnames query fail: %s", err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ifname string
		err = rows.Scan(&ifname)
		if err != nil {
			log.Errorf("GetUsedIfnames scan fail: %s", err)
			return nil
		}
		used[ifname] = true
	}
	return used
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

func (manager *SNetworkManager) SyncNetworks(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire, nets []cloudprovider.ICloudNetwork, syncOwnerId mcclient.IIdentityProvider) ([]SNetwork, []cloudprovider.ICloudNetwork, compare.SyncResult) {
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
		err = commondb[i].SyncWithCloudNetwork(ctx, userCred, commonext[i], syncOwnerId)
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
		new, err := manager.newFromCloudNetwork(ctx, userCred, added[i], wire, syncOwnerId)
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

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.NETWORK_STATUS_UNKNOWN, "Sync to remove")
	} else {
		err = self.RealDelete(ctx, userCred)

	}
	return err
}

func (self *SNetwork) SyncWithCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork, syncOwnerId mcclient.IIdentityProvider) error {
	vpc := self.GetWire().getVpc()
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

	return nil
}

func (manager *SNetworkManager) newFromCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork, wire *SWire, syncOwnerId mcclient.IIdentityProvider) (*SNetwork, error) {
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
	net.IsPublic = extNet.GetIsPublic()
	net.PublicScope = string(extNet.GetPublicScope())

	net.AllocTimoutSeconds = extNet.GetAllocTimeoutSeconds()

	err = manager.TableSpec().Insert(&net)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}

	vpc := wire.getVpc()
	SyncCloudProject(userCred, &net, syncOwnerId, extNet, vpc.ManagerId)

	db.OpsLog.LogEvent(&net, db.ACT_CREATE, net.GetShortDesc(ctx), userCred)

	return &net, nil
}

func (self *SNetwork) IsAddressInRange(address netutils.IPV4Addr) bool {
	return self.getIPRange().Contains(address)
}

func (self *SNetwork) isAddressUsed(address string) (bool, error) {
	managers := []db.IModelManager{
		GuestnetworkManager,
		GroupnetworkManager,
		HostnetworkManager,
		ReservedipManager,
		LoadbalancernetworkManager,
	}
	for _, manager := range managers {
		q := manager.Query().Equals("ip_addr", address).Equals("network_id", self.Id)
		cnt, err := q.CountWithError()
		if err != nil {
			return false, err
		}
		if cnt > 0 {
			return true, nil
		}
	}
	return false, nil
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

func (manager *SNetworkManager) allNetworksQ(providers []string, brands []string, cloudEnv string, rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
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
	return AttachUsageQuery(q, hosts, nil, nil, providers, brands, cloudEnv, rangeObj)
}

func (manager *SNetworkManager) totalPortCountQ(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, providers []string, brands []string, cloudEnv string, rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
	q := manager.allNetworksQ(providers, brands, cloudEnv, rangeObj)
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
	rangeObj db.IStandaloneModel,
) NetworkPortStat {
	nets := make([]SNetwork, 0)
	err := manager.totalPortCountQ(scope, userCred, providers, brands, cloudEnv, rangeObj).All(&nets)
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
		if net.IsOwner(userCred) || net.IsPublic || db.IsAdminAllowGet(userCred, net) ||
			utils.IsInStringArray(userCred.GetProjectId(), net.GetSharedProjects()) {
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

func isExitNetworkInfo(netConfig *api.NetworkConfig) bool {
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

func (self *SNetwork) getZone() *SZone {
	wire := self.GetWire()
	if wire != nil {
		return wire.GetZone()
	}
	return nil
}

func (self *SNetwork) getVpc() *SVpc {
	wire := self.GetWire()
	if wire != nil {
		return wire.getVpc()
	}
	return nil
}

func (self *SNetwork) getRegion() *SCloudregion {
	wire := self.GetWire()
	if wire != nil {
		return wire.getRegion()
	}
	return nil
}

func (self *SNetwork) GetPorts() int {
	return self.getIPRange().AddressCount()
}

func (self *SNetwork) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	wire := self.GetWire()
	zone := self.getZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.Name), "zone")
		extra.Add(jsonutils.NewString(zone.Id), "zone_id")
	}
	extra.Add(jsonutils.NewString(wire.Name), "wire")
	if self.IsExitNetwork() {
		extra.Add(jsonutils.JSONTrue, "exit")
	} else {
		extra.Add(jsonutils.JSONFalse, "exit")
	}
	extra.Add(jsonutils.NewInt(int64(self.GetPorts())), "ports")
	portsUsed, _ := self.GetTotalNicCount()
	extra.Add(jsonutils.NewInt(int64(portsUsed)), "ports_used")
	vnics, _ := self.GetGuestnicsCount()
	extra.Add(jsonutils.NewInt(int64(vnics)), "vnics")
	bmVnics, _ := self.GetBaremetalNicsCount()
	extra.Add(jsonutils.NewInt(int64(bmVnics)), "bm_vnics")
	lbVnics, _ := self.GetLoadbalancerIpsCount()
	extra.Add(jsonutils.NewInt(int64(lbVnics)), "lb_vnics")
	eips, _ := self.GetEipsCount()
	extra.Add(jsonutils.NewInt(int64(eips)), "eip_vnics")
	groupVnics, _ := self.GetGroupNicsCount()
	extra.Add(jsonutils.NewInt(int64(groupVnics)), "group_vnics")
	reserveVnics, _ := self.GetReservedNicsCount()
	extra.Add(jsonutils.NewInt(int64(reserveVnics)), "reserve_vnics")

	vpc := self.getVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.GetId()), "vpc_id")
		extra.Add(jsonutils.NewString(vpc.GetName()), "vpc")
		if len(vpc.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(vpc.GetExternalId()), "vpc_ext_id")
		}
	}
	routes := self.GetRoutes()
	if len(routes) > 0 {
		extra.Add(jsonutils.Marshal(routes), "routes")
	}

	info := vpc.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))
	extra = GetSchedtagsDetailsToResource(self, ctx, extra)

	return extra
}

func (self *SNetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = self.getMoreDetails(ctx, extra)
	return extra, nil
}

func (self *SNetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(ctx, extra)
	return extra
}

func (self *SNetwork) AllowPerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "reserve-ip")
}

func (self *SNetwork) PerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ips, err := data.GetArray("ips")
	if err != nil {
		if data.Contains("ip") {
			ip, _ := data.Get("ip")
			ips = []jsonutils.JSONObject{ip}
		} else {
			return nil, httperrors.NewMissingParameterError("ips")
		}
	}
	notes, err := data.GetString("notes")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("ips")
	}
	for _, ip := range ips {
		ipstr, _ := ip.GetString()
		ipAddr, err := netutils.NewIPV4Addr(ipstr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("not a valid ip address %s: %s", ipstr, err)
		}
		if !self.IsAddressInRange(ipAddr) {
			return nil, httperrors.NewInputParameterError("Address %s not in network", ipstr)
		}
		used, err := self.isAddressUsed(ipstr)
		if err != nil {
			return nil, httperrors.NewInternalServerError("isAddressUsed fail %s", err)
		}
		if used {
			return nil, httperrors.NewConflictError("Address %s has been used", ipstr)
		}
		err = ReservedipManager.ReserveIP(userCred, self, ipstr, notes)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (self *SNetwork) AllowPerformReleaseReservedIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "release-reserved-ip")
}

func (self *SNetwork) PerformReleaseReservedIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ipstr, _ := data.GetString("ip")
	if len(ipstr) == 0 {
		return nil, httperrors.NewInputParameterError("Reserved ip to release must be provided")
	}
	rip := ReservedipManager.GetReservedIP(self, ipstr)
	if rip == nil {
		return nil, httperrors.NewInvalidStatusError("Address %s not reserved", ipstr)
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

func (manager *SNetworkManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	prefixStr, _ := data.GetString("guest_ip_prefix")
	var maskLen64 int64
	var err error
	var startIp, endIp netutils.IPV4Addr
	if len(prefixStr) > 0 {
		prefix, err := netutils.NewIPV4Prefix(prefixStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("ip_prefix error: %s", err)
		}
		iprange := prefix.ToIPRange()
		startIp = iprange.StartIp().StepUp()
		endIp = iprange.EndIp().StepDown()
		maskLen64 = int64(prefix.MaskLen)
	} else {
		ipStartStr, _ := data.GetString("guest_ip_start")
		ipEndStr, _ := data.GetString("guest_ip_end")
		startIp, err = netutils.NewIPV4Addr(ipStartStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid start ip: %s %s", ipStartStr, err)
		}
		endIp, err = netutils.NewIPV4Addr(ipEndStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid end ip: %s %s", ipEndStr, err)
		}
		if startIp > endIp {
			tmp := startIp
			startIp = endIp
			endIp = tmp
		}
		maskLen64, _ = data.Int("guest_ip_mask")
	}
	if !isValidMaskLen(maskLen64) {
		return nil, httperrors.NewInputParameterError("Invalid masklen %d", maskLen64)
	}
	data.Add(jsonutils.NewInt(maskLen64), "guest_ip_mask")
	data.Add(jsonutils.NewString(startIp.String()), "guest_ip_start")
	data.Add(jsonutils.NewString(endIp.String()), "guest_ip_end")

	for _, key := range []string{"guest_gateway", "guest_dns", "guest_dhcp"} {
		ipStr, _ := data.GetString(key)
		if len(ipStr) > 0 && !regutils.MatchIPAddr(ipStr) {
			return nil, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipStr)
		}
	}

	nets := manager.getAllNetworks("")
	if nets == nil {
		return nil, httperrors.NewInternalServerError("query all networks fail")
	}

	if isOverlapNetworks(nets, startIp, endIp) {
		return nil, httperrors.NewInputParameterError("Conflict address space with existing networks")
	}

	wireStr := jsonutils.GetAnyString(data, []string{"wire", "wire_id"})
	if len(wireStr) > 0 {
		wireObj, err := WireManager.FetchByIdOrName(userCred, wireStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("wire %s not found", wireStr)
			} else {
				return nil, httperrors.NewInternalServerError("query wire %s error %s", wireStr, err)
			}
		}
		data.Add(jsonutils.NewString(wireObj.GetId()), "wire_id")
	} else {
		zoneStr := jsonutils.GetAnyString(data, []string{"zone", "zone_id"})
		if len(zoneStr) > 0 {
			vpcStr := jsonutils.GetAnyString(data, []string{"vpc", "vpc_id"})
			if len(vpcStr) > 0 {
				zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewNotFoundError("zone %s not found", zoneStr)
					} else {
						return nil, httperrors.NewInternalServerError("query zone %s error %s", zoneStr, err)
					}
				}
				vpcObj, err := VpcManager.FetchByIdOrName(userCred, vpcStr)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewNotFoundError("vpc %s not found", vpcStr)
					} else {
						return nil, httperrors.NewInternalServerError("query vpc %s error %s", vpcStr, err)
					}
				}
				vpc := vpcObj.(*SVpc)
				zone := zoneObj.(*SZone)
				region := zone.GetRegion()
				if region == nil {
					return nil, httperrors.NewInternalServerError("zone %s related region not found", zone.Id)
				}

				// 华为云,ucloud wire zone_id 为空
				var wires []SWire
				if utils.IsInStringArray(region.Provider, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_UCLOUD}) {
					wires, err = WireManager.getWiresByVpcAndZone(vpc, nil)
				} else {
					wires, err = WireManager.getWiresByVpcAndZone(vpc, zone)
				}

				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewNotFoundError("wire not found for zone %s and vpc %s", zoneStr, vpcStr)
					} else {
						return nil, httperrors.NewInternalServerError("query wire for zone %s and vpc %s error %s", zoneStr, vpcStr, err)
					}
				}
				if len(wires) == 0 {
					return nil, httperrors.NewNotFoundError("wire not found for zone %s and vpc %s", zoneStr, vpcStr)
				} else if len(wires) > 1 {
					return nil, httperrors.NewConflictError("more than 1 wire found for zone %s and vpc %s", zoneStr, vpcStr)
				} else {
					data.Add(jsonutils.NewString(wires[0].Id), "wire_id")
				}
			} else {
				return nil, httperrors.NewInputParameterError("No either wire or vpc provided")
			}
		} else {
			return nil, httperrors.NewInvalidStatusError("No either wire or zone provided")
		}
	}

	wireId, _ := data.GetString("wire_id")
	if len(wireId) == 0 {
		return nil, httperrors.NewMissingParameterError("wire_id")
	}
	wire := WireManager.FetchWireById(wireId)
	if wire == nil {
		return nil, httperrors.NewResourceNotFoundError("wire %s not found", wireId)
	}
	vpc := wire.getVpc()
	if vpc == nil {
		return nil, httperrors.NewInputParameterError("no valid vpc ???")
	}

	if vpc.Status != api.VPC_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("VPC not ready")
	}

	vpcRanges := vpc.getIPRanges()

	netRange := netutils.NewIPV4AddrRange(startIp, endIp)

	inRange := false
	for _, vpcRange := range vpcRanges {
		if vpcRange.ContainsRange(netRange) {
			inRange = true
			break
		}
	}

	if !inRange {
		return nil, httperrors.NewInputParameterError("Network not in range of VPC cidrblock %s", vpc.CidrBlock)
	}

	serverTypeStr, _ := data.GetString("server_type")
	if len(serverTypeStr) == 0 {
		serverTypeStr = api.NETWORK_TYPE_GUEST
	} else if !utils.IsInStringArray(serverTypeStr, ALL_NETWORK_TYPES) {
		return nil, httperrors.NewInputParameterError("Invalid server_type: %s", serverTypeStr)
	}
	data.Add(jsonutils.NewString(serverTypeStr), "server_type")

	return manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (self *SNetwork) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var startIp, endIp netutils.IPV4Addr
	var err error

	ipStartStr, _ := data.GetString("guest_ip_start")
	ipEndStr, _ := data.GetString("guest_ip_end")

	if len(ipStartStr) > 0 || len(ipEndStr) > 0 {
		if self.isManaged() {
			return nil, httperrors.NewForbiddenError("Cannot update a managed network")
		}

		if len(ipStartStr) > 0 {
			startIp, err = netutils.NewIPV4Addr(ipStartStr)
			if err != nil {
				return nil, httperrors.NewInputParameterError("Invalid start ip: %s %s", ipStartStr, err)
			}
		} else {
			startIp, _ = netutils.NewIPV4Addr(self.GuestIpStart)
		}
		if len(ipEndStr) > 0 {
			endIp, err = netutils.NewIPV4Addr(ipEndStr)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid end ip: %s %s", ipEndStr, err)
			}
		} else {
			endIp, _ = netutils.NewIPV4Addr(self.GuestIpEnd)
		}

		if startIp > endIp {
			tmp := startIp
			startIp = endIp
			endIp = tmp
		}

		nets := NetworkManager.getAllNetworks(self.Id)
		if nets == nil {
			return nil, httperrors.NewInternalServerError("query all networks fail")
		}

		if isOverlapNetworks(nets, startIp, endIp) {
			return nil, httperrors.NewInputParameterError("Conflict address space with existing networks")
		}

		vpc := self.GetVpc()

		vpcRanges := vpc.getIPRanges()

		netRange := netutils.NewIPV4AddrRange(startIp, endIp)

		inRange := false
		for _, vpcRange := range vpcRanges {
			if vpcRange.ContainsRange(netRange) {
				inRange = true
				break
			}
		}

		if !inRange {
			return nil, httperrors.NewInputParameterError("Network not in range of VPC cidrblock %s", vpc.CidrBlock)
		}

		usedMap := self.GetUsedAddresses()
		for usedIpStr := range usedMap {
			usedIp, _ := netutils.NewIPV4Addr(usedIpStr)
			if !netRange.Contains(usedIp) {
				return nil, httperrors.NewInputParameterError("Address been assigned out of new range")
			}
		}

		data.Add(jsonutils.NewString(startIp.String()), "guest_ip_start")
		data.Add(jsonutils.NewString(endIp.String()), "guest_ip_end")

	}

	if data.Contains("guest_ip_mask") {
		if self.isManaged() {
			return nil, httperrors.NewForbiddenError("Cannot update a managed network")
		}

		maskLen64, _ := data.Int("guest_ip_mask")
		if !isValidMaskLen(maskLen64) {
			return nil, httperrors.NewInputParameterError("Invalid masklen %d", maskLen64)
		}
	}

	for _, key := range []string{"guest_gateway", "guest_dns", "guest_dhcp"} {
		ipStr, _ := data.GetString(key)
		if len(ipStr) > 0 {
			if self.isManaged() {
				return nil, httperrors.NewForbiddenError("Cannot update a managed network")
			} else if !regutils.MatchIPAddr(ipStr) {
				return nil, httperrors.NewInputParameterError("%s: Invalid IP address %s", key, ipStr)
			}
		}
	}

	return self.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SNetworkManager) getAllNetworks(excludeId string) []SNetwork {
	nets := make([]SNetwork, 0)
	q := manager.Query()
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

func (self *SNetwork) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if db.IsAdminAllowCreate(userCred, self.GetModelManager()) && ownerId.GetProjectId() == userCred.GetProjectId() && self.ServerType == api.NETWORK_TYPE_GUEST {
		self.IsPublic = true
		self.PublicScope = string(rbacutils.ScopeDomain)
	} else {
		self.IsPublic = false
		self.PublicScope = string(rbacutils.ScopeNone)
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

func (manager *SNetworkManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()

	if query.Contains("ip") {
		// ipv4 only
		ip, err := query.GetString("ip")
		if err != nil {
			return nil, httperrors.NewInputParameterError("Get ip fail")
		}
		ipIa, err := parseIpToIntArray(ip)
		if err != nil {
			return nil, err
		}

		ipFilter := func(obj jsonutils.JSONObject) (bool, error) {
			guestIpStart, err := obj.GetString("guest_ip_start")
			if err != nil {
				return false, httperrors.NewInternalServerError("Get guest ip start error %s", err)
			}
			guestIpEnd, err := obj.GetString("guest_ip_end")
			if err != nil {
				return false, httperrors.NewInternalServerError("Get guest ip end error %s", err)
			}
			ipStartIa, err := parseIpToIntArray(guestIpStart)
			if err != nil {
				return false, httperrors.NewInternalServerError("Parse guest ip start error %s", err)
			}
			ipEndIa, err := parseIpToIntArray(guestIpEnd)
			if err != nil {
				return false, httperrors.NewInternalServerError("Parse guest ip end error %s", err)
			}
			for i := 0; i < len(ipIa); i++ {
				if ipIa[i] < ipStartIa[i] || ipIa[i] > ipEndIa[i] {
					return false, nil
				}
			}
			return true, nil
		}

		filters.Append(ipFilter)
	}
	return filters, nil
}

func (manager *SNetworkManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error

	q, err = managedResourceFilterByAccount(q, query, "wire_id", func() *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()

		subq := wires.Query(wires.Field("id"))
		subq = subq.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		return subq
	})
	if err != nil {
		return nil, err
	}

	q = managedResourceFilterByCloudType(q, query, "wire_id", func() *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		subq := wires.Query(wires.Field("id"))
		subq = subq.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		return subq
	})

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	zoneStr, _ := query.GetString("zone")
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("Zone %s not found", zoneStr)
		}
		zone := zoneObj.(*SZone)
		region := zone.GetRegion()
		if utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
			wires := WireManager.Query().SubQuery()
			vpcs := VpcManager.Query().SubQuery()

			sq := wires.Query(wires.Field("id")).
				Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
				Filter(sqlchemy.Equals(vpcs.Field("cloudregion_id"), region.Id))
			q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
		} else {
			sq := WireManager.Query("id").Equals("zone_id", zoneObj.GetId())
			q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
		}
	}

	vpcStr, _ := query.GetString("vpc")
	if len(vpcStr) > 0 {
		vpcObj, err := VpcManager.FetchByIdOrName(userCred, vpcStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("VPC %s not found", vpcStr)
		}
		sq := WireManager.Query("id").Equals("vpc_id", vpcObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}

	regionStr := jsonutils.GetAnyString(query, []string{"region_id", "region", "cloudregion_id", "cloudregion"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("cloud region %s not found", regionStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		sq := wires.Query(wires.Field("id")).
			Join(vpcs, sqlchemy.AND(
				sqlchemy.Equals(vpcs.Field("cloudregion_id"), region.GetId()),
				sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))))
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		wires := WireManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		regions := CloudregionManager.Query().SubQuery()

		sq := wires.Query(wires.Field("id")).
			Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
			Join(zones, sqlchemy.OR(sqlchemy.Equals(wires.Field("zone_id"), zones.Field("id")), sqlchemy.IsNullOrEmpty(wires.Field("zone_id")))).
			Join(regions, sqlchemy.Equals(zones.Field("cloudregion_id"), regions.Field("id"))).
			Filter(sqlchemy.AND(
				sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE),
				sqlchemy.Equals(zones.Field("status"), api.ZONE_ENABLE),
				sqlchemy.Equals(regions.Field("status"), api.CLOUD_REGION_STATUS_INSERVER),
			))
		q = q.In("wire_id", sq.SubQuery()).Equals("status", api.NETWORK_STATUS_AVAILABLE)
	}

	hostStr, _ := query.GetString("host")
	if len(hostStr) > 0 {
		hostObj, err := HostManager.FetchByIdOrName(userCred, hostStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), hostStr)
		}
		sq := HostwireManager.Query("wire_id").Equals("host_id", hostObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}

	cityStr, _ := query.GetString("city")
	if len(cityStr) > 0 {
		regions := CloudregionManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		sq := wires.Query(wires.Field("id")).
			Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
			Join(regions, sqlchemy.Equals(regions.Field("id"), vpcs.Field("cloudregion_id"))).
			Filter(sqlchemy.Equals(regions.Field("city"), cityStr))
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}

	return q, nil
}

func (manager *SNetworkManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "account":
		vpcs := VpcManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudaccounts := CloudaccountManager.Query("name", "id").Distinct().SubQuery()
		q = q.Join(wires, sqlchemy.Equals(q.Field("wire_id"), wires.Field("id")))
		q = q.Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))
		q = q.Join(cloudproviders, sqlchemy.Equals(vpcs.Field("manager_id"), cloudproviders.Field("id")))
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), cloudaccounts.Field("id")))
		q.GroupBy(cloudaccounts.Field("name"))
		q.AppendField(cloudaccounts.Field("name", "account"))
	default:
		return nil, httperrors.NewBadRequestError("unsupport field %s", field)
	}

	return q, nil
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
		if len(n.ExternalId) == 0 && len(n.WireId) > 0 && n.Status == api.NETWORK_STATUS_INIT {
			db.Update(&n, func() error {
				n.Status = api.NETWORK_STATUS_AVAILABLE
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

func (self *SNetwork) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	vpc := self.GetVpc()
	if vpc != nil && len(vpc.ExternalId) > 0 {
		provider := vpc.GetCloudprovider()
		if provider != nil && provider.Enabled {
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

func (self *SNetwork) PerformMerge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	target, err := data.GetString("target")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("target")
	}
	iNet, err := NetworkManager.FetchByIdOrName(userCred, target)
	if err == sql.ErrNoRows {
		err = httperrors.NewNotFoundError("Network %s not found", target)
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
		err = httperrors.NewInputParameterError("Invalid Target Network: %s", target)
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

func (self *SNetwork) PerformSplit(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	splitIp, err := data.GetString("split_ip")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("split_ip")
	}
	name, _ := data.GetString("name")

	if !regutils.MatchIPAddr(splitIp) {
		return nil, httperrors.NewInputParameterError("Invalid IP %s", splitIp)
	}
	if splitIp == self.GuestIpStart {
		return nil, httperrors.NewInputParameterError("Split IP %s is the start ip", splitIp)
	}

	iSplitIp, err := netutils.NewIPV4Addr(splitIp)
	if err != nil {
		return nil, err
	}
	if !self.IsAddressInRange(iSplitIp) {
		return nil, httperrors.NewInputParameterError("Split IP %s out of range", splitIp)
	}

	lockman.LockClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))
	defer lockman.ReleaseClass(ctx, NetworkManager, db.GetLockClassKey(NetworkManager, userCred))

	if len(name) > 0 {
		if err := db.NewNameValidator(NetworkManager, userCred, name); err != nil {
			return nil, httperrors.NewInputParameterError("Duplicate name %s", name)
		}
	} else {
		newName, err := db.GenerateName(NetworkManager, userCred, fmt.Sprintf("%s#", self.Name))
		if err != nil {
			return nil, httperrors.NewInternalServerError("GenerateName fail %s", err)
		}
		name = newName
	}

	network := &SNetwork{}
	network.Name = name
	network.GuestIpStart = splitIp
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

	note := map[string]string{"split_ip": splitIp, "end_ip": network.GuestIpEnd}
	db.OpsLog.LogEvent(self, db.ACT_SPLIT, note, userCred)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_SPLIT, note, userCred, true)
	db.OpsLog.LogEvent(network, db.ACT_CREATE, map[string]string{"network": self.Id}, userCred)
	return nil, nil
}

func (manager *SNetworkManager) AllowPerformTryCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "try-create-network")
}

func (manager *SNetworkManager) PerformTryCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ip, err := data.GetString("ip")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("ip")
	}
	ipV4, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		return nil, httperrors.NewInputParameterError("ip")
	}
	mask, err := data.Int("mask")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("mask")
	}
	serverType, err := data.GetString("server_type")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("server_type")
	}
	if serverType != api.NETWORK_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Only support server type %s", api.NETWORK_TYPE_BAREMETAL)
	}
	if !jsonutils.QueryBoolean(data, "is_on_premise", false) {
		return nil, httperrors.NewBadRequestError("Only support on premise network")
	}

	var (
		ipV4NetAddr = ipV4.NetAddr(int8(mask))
		nm          *SNetwork
		matched     bool
	)

	q := NetworkManager.Query().Equals("server_type", serverType).Equals("guest_ip_mask", mask)
	q = managedResourceFilterByCloudType(q, query, "wire_id", func() *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		subq := wires.Query(wires.Field("id"))
		subq = subq.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		return subq
	})

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
	} else {
		ret.Set("find_matched", jsonutils.JSONTrue)
		ret.Set("wire_id", jsonutils.NewString(nm.WireId))
	}
	if !matched {
		log.Infof("Find same subnet network %s %s/%d", nm.Name, nm.GuestGateway, nm.GuestIpMask)
		newNetwork := new(SNetwork)
		newNetwork.SetModelManager(NetworkManager, newNetwork)
		newNetwork.GuestIpStart = ip
		newNetwork.GuestIpEnd = ip
		newNetwork.GuestGateway = nm.GuestGateway
		newNetwork.GuestIpMask = int8(mask)
		newNetwork.GuestDns = nm.GuestDns
		newNetwork.GuestDhcp = nm.GuestDhcp
		newNetwork.WireId = nm.WireId
		newNetwork.ServerType = serverType
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
		err = newNetwork.CustomizeCreate(ctx, userCred, userCred, query, data)
		if err != nil {
			return nil, err
		}
		newNetwork.PostCreate(ctx, userCred, userCred, query, data)
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

func (manager *SNetworkManager) IsValidOnPremiseNetworkIP(ipStr string) bool {
	net, _ := manager.GetOnPremiseNetworkOfIP(ipStr, "", tristate.None)
	if net != nil {
		return true
	}
	return false
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

func (network *SNetwork) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, err := network.SSharableVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	network.ClearSchedDescCache()
	return ret, nil
}
