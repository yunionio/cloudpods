package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	// # DEFAULT_BANDWIDTH = options.default_bandwidth
	MAX_BANDWIDTH = 100000

	SERVER_TYPE_GUEST     = "guest"
	SERVER_TYPE_BAREMETAL = "baremetal"
	SERVER_TYPE_CONTAINER = "container"

	STATIC_ALLOC = "static"

	MAX_NETWORK_NAME_LEN = 11

	EXTRA_DNS_UPDATE_TARGETS = "__extra_dns_update_targets"

	NETWORK_STATUS_INIT          = "init"
	NETWORK_STATUS_PENDING       = "pending"
	NETWORK_STATUS_AVAILABLE     = "available"
	NETWORK_STATUS_FAILED        = "failed"
	NETWORK_STATUS_UNKNOWN       = "unknown"
	NETWORK_STATUS_START_DELETE  = "start_delete"
	NETWORK_STATUS_DELETING      = "deleting"
	NETWORK_STATUS_DELETED       = "deleted"
	NETWORK_STATUS_DELETE_FAILED = "delete_failed"
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
	NetworkManager = &SNetworkManager{SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(SNetwork{}, "networks_tbl", "network", "networks")}
	NetworkManager.NameLength = 9
	NetworkManager.NameRequireAscii = true
}

type SNetwork struct {
	db.SSharableVirtualResourceBase

	GuestIpStart string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	GuestIpEnd   string `width:"16" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	GuestIpMask  int8   `nullable:"false" list:"user" update:"user" create:"required"`                            // Column(TINYINT, nullable=False)
	GuestGateway string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`  // Column(VARCHAR(16, charset='ascii'), nullable=True)
	GuestDns     string `width:"16" charset:"ascii" nullable:"true" get:"user" update:"user" create:"optional"`   // Column(VARCHAR(16, charset='ascii'), nullable=True)
	GuestDhcp    string `width:"16" charset:"ascii" nullable:"true" get:"user" update:"user" create:"optional"`   // Column(VARCHAR(16, charset='ascii'), nullable=True)

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

	ServerType string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AllocPolicy string `width:"16" charset:"ascii" nullable:"true" get:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AllocTimoutSeconds int `default:"0" nullable:"true" get:"admin"`
}

func (manager *SNetworkManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{WireManager}
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
	return userCred.IsSystemAdmin()
}

func (self *SNetwork) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetTotalNicCount() > 0 {
		return httperrors.NewNotEmptyError("not an empty network")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SNetwork) GetTotalNicCount() int {
	return self.GetGuestnicsCount() + self.GetGroupNicsCount() + self.GetBaremetalNicsCount() + self.GetReservedNicsCount()
}

func (self *SNetwork) GetGuestnicsCount() int {
	return GuestnetworkManager.Query().Equals("network_id", self.Id).IsFalse("virtual").Count()
}

func (self *SNetwork) GetGroupNicsCount() int {
	return GroupnetworkManager.Query().Equals("network_id", self.Id).Count()
}

func (self *SNetwork) GetBaremetalNicsCount() int {
	return HostnetworkManager.Query().Equals("network_id", self.Id).Count()
}

func (self *SNetwork) GetReservedNicsCount() int {
	return ReservedipManager.Query().Equals("network_id", self.Id).Count()
}

func (self *SNetwork) GetUsedAddresses() map[string]bool {
	used := make(map[string]bool)

	for _, tbl := range []*sqlchemy.SSubQuery{
		GuestnetworkManager.Query().SubQuery(),
		GroupnetworkManager.Query().SubQuery(),
		HostnetworkManager.Query().SubQuery(),
		ReservedipManager.Query().SubQuery(),
		LoadbalancernetworkManager.Query().SubQuery(),
	} {
		q := tbl.Query(tbl.Field("ip_addr")).Equals("network_id", self.Id)
		rows, err := q.Rows()
		if err != nil {
			log.Errorf("GetUsedAddresses query fail: %s", err)
			return nil
		}
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
			return "", httperrors.NewInsufficientResourceError(fmt.Sprintf("Reserved address %s not found", candidate))
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

type DNSUpdateKeySecret struct {
	Key    string
	Secret string
}

func (self *SNetwork) getDnsUpdateTargets() map[string][]DNSUpdateKeySecret {
	targets := make(map[string][]DNSUpdateKeySecret)
	targetsJson := self.GetMetadataJson(EXTRA_DNS_UPDATE_TARGETS, nil)
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
	nets := make([]SNetwork, 0)
	q := manager.Query().Equals("wire_id", wire.Id)
	err := db.FetchModelObjects(manager, q, &nets)
	if err != nil {
		log.Errorf("getNetworkByWire fail %s", err)
		return nil, err
	}
	return nets, nil
}

func (manager *SNetworkManager) SyncNetworks(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire, nets []cloudprovider.ICloudNetwork) ([]SNetwork, []cloudprovider.ICloudNetwork, compare.SyncResult) {
	localNets := make([]SNetwork, 0)
	remoteNets := make([]cloudprovider.ICloudNetwork, 0)
	syncResult := compare.SyncResult{}

	dbNets, err := manager.getNetworksByWire(wire)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
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
		/*err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			syncResult.DeleteError(err)
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}*/
		err = removed[i].SetStatus(userCred, NETWORK_STATUS_UNKNOWN, "Sync to remove")
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudNetwork(userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localNets = append(localNets, commondb[i])
			remoteNets = append(remoteNets, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudNetwork(userCred, added[i], wire)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localNets = append(localNets, *new)
			remoteNets = append(remoteNets, added[i])
			syncResult.Add()
		}
	}

	return localNets, remoteNets, syncResult
}

func (self *SNetwork) SyncWithCloudNetwork(userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		extNet.Refresh()
		self.Name = extNet.GetName()
		self.Status = extNet.GetStatus()
		self.GuestIpStart = extNet.GetIpStart()
		self.GuestIpEnd = extNet.GetIpEnd()
		self.GuestIpMask = extNet.GetIpMask()
		self.GuestGateway = extNet.GetGateway()
		self.ServerType = extNet.GetServerType()
		self.IsPublic = extNet.GetIsPublic()

		self.AllocTimoutSeconds = extNet.GetAllocTimeoutSeconds()

		self.ProjectId = userCred.GetProjectId()
		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudNetwork error %s", err)
	}
	return err
}

func (manager *SNetworkManager) newFromCloudNetwork(userCred mcclient.TokenCredential, extNet cloudprovider.ICloudNetwork, wire *SWire) (*SNetwork, error) {
	net := SNetwork{}
	net.SetModelManager(manager)

	net.Name = extNet.GetName()
	net.Status = extNet.GetStatus()
	net.ExternalId = extNet.GetGlobalId()
	net.WireId = wire.Id
	net.GuestIpStart = extNet.GetIpStart()
	net.GuestIpEnd = extNet.GetIpEnd()
	net.GuestIpMask = extNet.GetIpMask()
	net.GuestGateway = extNet.GetGateway()
	net.ServerType = extNet.GetServerType()
	net.IsPublic = extNet.GetIsPublic()

	net.AllocTimoutSeconds = extNet.GetAllocTimeoutSeconds()

	net.ProjectId = userCred.GetProjectId()

	err := manager.TableSpec().Insert(&net)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}
	return &net, nil
}

func (self *SNetwork) isAddressInRange(address netutils.IPV4Addr) bool {
	return self.getIPRange().Contains(address)
}

func (self *SNetwork) isAddressUsed(address string) bool {
	managers := []db.IModelManager{
		GuestnetworkManager,
		GroupnetworkManager,
		HostnetworkManager,
		ReservedipManager,
		LoadbalancernetworkManager,
	}
	for _, manager := range managers {
		q := manager.Query().Equals("ip_addr", address).Equals("network_id", self.Id)
		if q.Count() > 0 {
			return true
		}
	}
	return false
}

func (manager *SNetworkManager) GetNetworkOfIP(ipAddr string, serverType string, isPublic tristate.TriState) (*SNetwork, error) {
	address, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return nil, err
	}
	q := manager.Query()
	if len(serverType) > 0 {
		q = q.Equals("server_type", serverType)
	}
	if isPublic.IsTrue() {
		q = q.IsTrue("is_public")
	} else if isPublic.IsFalse() {
		q = q.IsFalse("is_public")
	}
	nets := make([]SNetwork, 0)
	err = db.FetchModelObjects(manager, q, &nets)
	if err != nil {
		return nil, err
	}
	for _, n := range nets {
		if n.isAddressInRange(address) {
			return &n, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (manager *SNetworkManager) allNetworksQ(rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
	networks := manager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()
	q := networks.Query().
		Join(hostwires, sqlchemy.AND(
			sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")),
			sqlchemy.IsFalse(hostwires.Field("deleted")))).
		Join(hosts, sqlchemy.AND(
			sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")),
			sqlchemy.IsFalse(hosts.Field("deleted")),
			sqlchemy.IsTrue(hosts.Field("enabled")),
			sqlchemy.OR(
				sqlchemy.Equals(hosts.Field("host_type"), HOST_TYPE_BAREMETAL),
				sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE))))
	return AttachUsageQuery(q, hosts, hostwires.Field("host_id"), nil, rangeObj)
}

func (manager *SNetworkManager) totalPortCountQ(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
	q := manager.allNetworksQ(rangeObj)
	networks := manager.Query().SubQuery()
	if userCred != nil && !userCred.IsSystemAdmin() {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(networks.Field("tenant_id"), userCred.GetProjectId()),
			sqlchemy.IsTrue(networks.Field("is_public"))))
	}
	return q
}

type NetworkPortStat struct {
	Count    int
	CountExt int
}

func (manager *SNetworkManager) TotalPortCount(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel) NetworkPortStat {
	nets := make([]SNetwork, 0)
	err := manager.totalPortCountQ(userCred, rangeObj).All(&nets)
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

type SNetworkConfig struct {
	Network  string
	Wire     string
	Exit     bool
	Private  bool
	Mac      string
	Address  string
	Address6 string
	Driver   string
	BwLimit  int
	Vip      bool
	Reserved bool
}

func parseNetworkInfo(userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SNetworkConfig, error) {
	netConfig := SNetworkConfig{}

	netJson, ok := info.(*jsonutils.JSONDict)
	if ok {
		err := netJson.Unmarshal(&netConfig)
		if err != nil {
			return nil, err
		}
		return &netConfig, nil
	}
	netStr, err := info.GetString()
	if err != nil {
		log.Errorf("invalid networkinfo format %s", err)
		return nil, err
	}
	parts := strings.Split(netStr, ":")
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if regutils.MatchIP4Addr(p) {
			netConfig.Address = p
		} else if regutils.MatchIP6Addr(p) {
			netConfig.Address6 = p
		} else if regutils.MatchCompactMacAddr(p) {
			netConfig.Mac = netutils.MacUnpackHex(p)
		} else if strings.HasPrefix(p, "wire=") {
			netConfig.Wire = p[len("wire="):]
		} else if p == "[random_exit]" {
			netConfig.Exit = true
		} else if p == "[random]" {
			netConfig.Exit = false
		} else if p == "[private]" {
			netConfig.Private = true
		} else if p == "[reserved]" {
			netConfig.Reserved = true
		} else if utils.IsInStringArray(p, []string{"virtio", "e1000", "vmxnet3"}) {
			netConfig.Driver = p
		} else if regutils.MatchSize(p) {
			bw, err := fileutils.GetSizeMb(p, 'M', 1000)
			if err != nil {
				return nil, err
			}
			netConfig.BwLimit = bw
		} else if p == "[vip]" {
			netConfig.Vip = true
		} else {
			netObj, err := NetworkManager.FetchByIdOrName(userCred.GetProjectId(), p)
			if err != nil {
				return nil, err
			}
			netConfig.Network = netObj.GetId()
		}
	}
	if netConfig.BwLimit == 0 {
		netConfig.BwLimit = options.Options.DefaultBandwidth
	}
	return &netConfig, nil
}

func (self *SNetwork) getFreeAddressCount() int {
	return self.getIPRange().AddressCount() - self.GetTotalNicCount()
}

func isValidNetworkInfo(userCred mcclient.TokenCredential, netConfig *SNetworkConfig) error {
	if len(netConfig.Network) > 0 {
		netObj, err := NetworkManager.FetchByIdOrName(userCred.GetProjectId(), netConfig.Network)
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
			if !net.isAddressInRange(ipAddr) {
				return httperrors.NewInputParameterError("Address %s not in range", netConfig.Address)
			}
			if netConfig.Reserved {
				if !userCred.IsSystemAdmin() {
					return httperrors.NewForbiddenError("Only system admin allowed to use reserved ip")
				}
				if ReservedipManager.GetReservedIP(net, netConfig.Address) == nil {
					return httperrors.NewInputParameterError("Address %s not reserved", netConfig.Address)
				}
			} else if net.isAddressUsed(netConfig.Address) {
				return httperrors.NewInputParameterError("Address %s has been used", netConfig.Address)
			}
		}
		if netConfig.BwLimit > MAX_BANDWIDTH {
			return httperrors.NewInputParameterError("Bandwidth limit cannot exceed %dMbps", MAX_BANDWIDTH)
		}
	}
	/* scheduler to the check
	else if ! netConfig.Vip {
		ct, ctExit := NetworkManager.to
	}
	*/
	return nil
}

func isExitNetworkInfo(netConfig *SNetworkConfig) bool {
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

func (self *SNetwork) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	wire := self.GetWire()
	extra.Add(jsonutils.NewString(wire.Name), "wire")
	if self.IsExitNetwork() {
		extra.Add(jsonutils.JSONTrue, "exit")
	} else {
		extra.Add(jsonutils.JSONFalse, "exit")
	}
	extra.Add(jsonutils.NewInt(int64(self.getIPRange().AddressCount())), "ports")
	extra.Add(jsonutils.NewInt(int64(self.GetGuestnicsCount())), "vnics")
	extra.Add(jsonutils.NewInt(int64(self.GetBaremetalNicsCount())), "bm_vnics")
	extra.Add(jsonutils.NewInt(int64(self.GetGroupNicsCount())), "group_vnics")
	extra.Add(jsonutils.NewInt(int64(self.GetReservedNicsCount())), "reserve_vnics")

	zone := self.getZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.GetId()), "zone_id")
		extra.Add(jsonutils.NewString(zone.GetName()), "zone")
		if len(zone.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(zone.GetExternalId()), "zone_external_id")
		}
	}

	region := self.getRegion()
	if region != nil {
		extra.Add(jsonutils.NewString(region.GetId()), "region_id")
		extra.Add(jsonutils.NewString(region.GetName()), "region")
		if len(region.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(region.GetExternalId()), "region_external_id")
		}
	}

	vpc := self.getVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.GetId()), "vpc_id")
		extra.Add(jsonutils.NewString(vpc.GetName()), "vpc")
		if len(vpc.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(vpc.GetExternalId()), "vpc_external_id")
		}
	}

	return extra
}

func (self *SNetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SNetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SNetwork) AllowPerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SNetwork) PerformReserveIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ipstr, _ := data.GetString("ip")
	notes, _ := data.GetString("notes")
	if len(ipstr) == 0 || len(notes) == 0 {
		return nil, httperrors.NewInputParameterError("both reserved ip and notes should be provided")
	}
	ipAddr, err := netutils.NewIPV4Addr(ipstr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("not a valid ip address %s: %s", ipstr, err)
	}
	if !self.isAddressInRange(ipAddr) {
		return nil, httperrors.NewInputParameterError("Address %s not in network", ipstr)
	}
	if self.isAddressUsed(ipstr) {
		return nil, httperrors.NewConflictError("Address %s has been used", ipstr)
	}
	err = ReservedipManager.ReserveIP(userCred, self, ipstr, notes)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (self *SNetwork) AllowPerformReleaseReservedIp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
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
	return self.IsOwner(userCred)
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

func (manager *SNetworkManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
		wireObj, err := WireManager.FetchByIdOrName(userCred.GetProjectId(), wireStr)
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
				zoneObj, err := ZoneManager.FetchByIdOrName(userCred.GetProjectId(), zoneStr)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewNotFoundError("zone %s not found", zoneStr)
					} else {
						return nil, httperrors.NewInternalServerError("query zone %s error %s", zoneStr, err)
					}
				}
				vpcObj, err := VpcManager.FetchByIdOrName(userCred.GetProjectId(), vpcStr)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewNotFoundError("vpc %s not found", vpcStr)
					} else {
						return nil, httperrors.NewInternalServerError("query vpc %s error %s", vpcStr, err)
					}
				}
				vpc := vpcObj.(*SVpc)
				zone := zoneObj.(*SZone)
				wires, err := WireManager.getWiresByVpcAndZone(vpc, zone)
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
		return nil, httperrors.NewInputParameterError("missing wire_id")
	}
	wire := WireManager.FetchWireById(wireId)
	if wire == nil {
		return nil, httperrors.NewInputParameterError("wire_id %s not valid", wireId)
	}
	vpc := wire.getVpc()
	if vpc == nil {
		return nil, httperrors.NewInputParameterError("no valid vpc ???")
	}

	if vpc.Status != VPC_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("VPC not ready")
	}

	vpcRange := vpc.getIPRange()

	netRange := netutils.NewIPV4AddrRange(startIp, endIp)

	if !vpcRange.ContainsRange(netRange) {
		return nil, httperrors.NewInputParameterError("Network not in range of VPC cidrblock %s", vpc.CidrBlock)
	}

	serverTypeStr, _ := data.GetString("server_type")
	if len(serverTypeStr) == 0 {
		serverTypeStr = SERVER_TYPE_GUEST
	} else if !sets.NewString(SERVER_TYPE_GUEST, SERVER_TYPE_BAREMETAL, SERVER_TYPE_CONTAINER).Has(serverTypeStr) {
		return nil, httperrors.NewInputParameterError("Invalid server_type: %s", serverTypeStr)
	}
	data.Add(jsonutils.NewString(serverTypeStr), "server_type")

	return manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
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

		vpcRange := vpc.getIPRange()

		netRange := netutils.NewIPV4AddrRange(startIp, endIp)

		if !vpcRange.ContainsRange(netRange) {
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

func (self *SNetwork) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if userCred.IsSystemAdmin() && ownerProjId == userCred.GetProjectId() {
		self.IsPublic = true
	} else {
		self.IsPublic = false
	}
	return self.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (self *SNetwork) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	wire := self.GetWire()
	if wire == nil {
		log.Errorf("cannot find wire???")
	} else {
		wire.clearHostSchedDescCache()
	}
	vpc := self.GetVpc()
	if vpc != nil && vpc.IsManaged() {
		task, err := taskman.TaskManager.NewTask(ctx, "NetworkCreateTask", self, userCred, nil, "", "", nil)
		if err != nil {
			log.Errorf("networkcreateTask create fail: %s", err)
		} else {
			task.ScheduleRun(nil)
		}
	} else {
		self.SetStatus(userCred, NETWORK_STATUS_AVAILABLE, "")
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
	self.SetStatus(userCred, NETWORK_STATUS_START_DELETE, "")
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
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(), userCred)
	self.SetStatus(userCred, NETWORK_STATUS_DELETED, "real delete")
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

func (manager *SNetworkManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	zoneStr, _ := query.GetString("zone")
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred.GetProjectId(), zoneStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("Zone %s not found", zoneStr)
		}
		sq := WireManager.Query("id").Equals("zone_id", zoneObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}
	vpcStr, _ := query.GetString("vpc")
	if len(vpcStr) > 0 {
		vpcObj, err := VpcManager.FetchByIdOrName(userCred.GetProjectId(), vpcStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("VPC %s not found", vpcStr)
		}
		sq := WireManager.Query("id").Equals("vpc_id", vpcObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), sq.SubQuery()))
	}
	regionStr := jsonutils.GetAnyString(query, []string{"region_id", "region", "cloudregion_id", "cloudregion"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(userCred.GetProjectId(), regionStr)
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
		if len(n.ExternalId) == 0 && len(n.WireId) > 0 && n.Status == NETWORK_STATUS_INIT {
			manager.TableSpec().Update(&n, func() error {
				n.Status = NETWORK_STATUS_AVAILABLE
				return nil
			})
		}
	}
	return nil
}

func (self *SNetwork) ValidateUpdateCondition(ctx context.Context) error {
	if len(self.ExternalId) > 0 {
		return httperrors.NewConflictError("Cannot update external resource")
	}
	return self.SSharableVirtualResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SNetwork) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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
