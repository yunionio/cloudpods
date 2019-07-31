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
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	MAX_IFNAME_SIZE = 13

	MAX_GUESTNIC_TO_SAME_NETWORK = 2
)

type SGuestnetworkManager struct {
	SGuestJointsManager
}

var GuestnetworkManager *SGuestnetworkManager

func init() {
	db.InitManager(func() {
		GuestnetworkManager = &SGuestnetworkManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestnetwork{},
				"guestnetworks_tbl",
				"guestnetwork",
				"guestnetworks",
				NetworkManager,
			),
		}
		GuestnetworkManager.SetVirtualObject(GuestnetworkManager)
		GuestnetworkManager.TableSpec().AddIndex(true, "ip_addr", "guest_id")
	})
}

type SGuestnetwork struct {
	SGuestJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" `             // Column(VARCHAR(36, charset='ascii'), nullable=False)
	MacAddr   string `width:"32" charset:"ascii" nullable:"false" list:"user"`              // Column(VARCHAR(32, charset='ascii'), nullable=False)
	IpAddr    string `width:"16" charset:"ascii" nullable:"false" list:"user"`              // Column(VARCHAR(16, charset='ascii'), nullable=True)
	Ip6Addr   string `width:"64" charset:"ascii" nullable:"true" list:"user"`               // Column(VARCHAR(64, charset='ascii'), nullable=True)
	Driver    string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	BwLimit   int    `nullable:"false" default:"0" list:"user"`                             // Column(Integer, nullable=False, default=0) # Mbps
	Index     int8   `nullable:"false" default:"0" list:"user" update:"user"`               // Column(TINYINT, nullable=False, default=0)
	Virtual   bool   `default:"false" list:"user"`                                          // Column(Boolean, default=False)
	Ifname    string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	TeamWith string `width:"32" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SGuestnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (joint *SGuestnetwork) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGuestnetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SGuestnetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGuestJointsBase.GetCustomizeColumns(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
}

func (self *SGuestnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SGuestJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return db.JointModelExtra(self, extra), nil
}

func (manager *SGuestnetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (gn *SGuestnetwork) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

const MAX_TRIES = 10

func (manager *SGuestnetworkManager) GenerateMac(netId string, suggestion string) (string, error) {
	for tried := 0; tried < MAX_TRIES; tried += 1 {
		var mac string
		if len(suggestion) > 0 && regutils.MatchMacAddr(suggestion) {
			mac = suggestion
			suggestion = ""
		} else {
			b := make([]byte, 4)
			_, err := rand.Read(b)
			if err != nil {
				log.Errorf("generate random mac failed: %s", err)
				continue
			}
			mac = fmt.Sprintf("00:22:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3])
		}
		q := manager.Query().Equals("mac_addr", mac)
		if len(netId) > 0 {
			q = q.Equals("network_id", netId)
		}
		cnt, err := q.CountWithError()
		if err != nil {
			log.Errorf("find mac %s error %s", mac, err)
			return "", err
		}
		if cnt == 0 {
			return mac, nil
		}
	}
	return "", fmt.Errorf("maximal retry reached")
}

func (manager *SGuestnetworkManager) newGuestNetwork(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, network *SNetwork,
	index int8, address string, mac string, driver string, bwLimit int, virtual bool, reserved bool,
	allocDir IPAddlocationDirection, requiredDesignatedIp bool, ifName string, teamWithMac string) (*SGuestnetwork, error) {

	gn := SGuestnetwork{}
	gn.SetModelManager(GuestnetworkManager, &gn)

	gn.GuestId = guest.Id
	gn.NetworkId = network.Id
	gn.Index = index
	gn.Virtual = virtual
	if len(driver) == 0 {
		driver = "virtio"
	}
	gn.Driver = driver
	if bwLimit >= 0 {
		gn.BwLimit = bwLimit
	}

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)

	macAddr, err := manager.GenerateMac(network.Id, mac)
	if err != nil {
		return nil, err
	}
	if len(macAddr) == 0 {
		log.Errorf("Mac address generate fails")
		return nil, fmt.Errorf("mac address generate fails")
	}
	gn.MacAddr = macAddr
	if !virtual {
		addrTable := network.GetUsedAddresses()
		recentAddrTable := manager.getRecentlyReleasedIPAddresses(network.Id, network.getAllocTimoutDuration())
		ipAddr, err := network.GetFreeIP(ctx, userCred, addrTable, recentAddrTable, address, allocDir, reserved)
		if err != nil {
			return nil, err
		}
		if len(address) > 0 && ipAddr != address && requiredDesignatedIp {
			return nil, fmt.Errorf("candidate ip %s is occupied!", address)
		}
		gn.IpAddr = ipAddr
	}
	ifTable := network.GetUsedIfnames()
	if len(ifName) > 0 {
		if _, ok := ifTable[ifName]; ok {
			ifName = ""
			log.Infof("ifname %s has been used, to release ...", ifName)
		}
	}
	if len(ifName) == 0 {
		ifName = gn.GetFreeIfname(network, ifTable)
	}
	gn.Ifname = ifName
	gn.TeamWith = teamWithMac
	err = manager.TableSpec().Insert(&gn)
	if err != nil {
		return nil, err
	}
	return &gn, nil
}

func (self *SGuestnetwork) getVirtualRand(width int, randomized bool) string {
	hash := md5.New()
	io.WriteString(hash, self.GuestId)
	io.WriteString(hash, self.NetworkId)
	if randomized {
		io.WriteString(hash, fmt.Sprintf("%d", time.Now().Unix()))
	}
	hex := fmt.Sprintf("%x", hash.Sum(nil))
	return hex[:width]
}

func (self *SGuestnetwork) generateIfname(network *SNetwork, virtual bool, randomized bool) string {
	pattern := regexp.MustCompile(`\W+`)
	nName := pattern.ReplaceAllString(network.Name, "")
	if len(nName) > MAX_IFNAME_SIZE-4 {
		nName = nName[:(MAX_IFNAME_SIZE - 4)]
	}
	if virtual {
		rand := self.getVirtualRand(3, randomized)
		return fmt.Sprintf("%s-%s", nName, rand)
	} else {
		ip, _ := netutils.NewIPV4Addr(self.IpAddr)
		cliaddr := ip.CliAddr(network.GuestIpMask)
		return fmt.Sprintf("%s-%d", nName, uint32(cliaddr))
	}
}

func (self *SGuestnetwork) GetFreeIfname(network *SNetwork, ifTable map[string]bool) string {
	ifname := self.generateIfname(network, self.Virtual, false)
	if _, exist := ifTable[ifname]; exist {
		if !self.Virtual {
			ifname = self.generateIfname(network, true, false)
		}
		for {
			if _, exist = ifTable[ifname]; exist {
				ifname = self.generateIfname(network, true, true)
			} else {
				break
			}
		}
	}
	return ifname
}

func (self *SGuestnetwork) GetGuest() *SGuest {
	guest, _ := GuestManager.FetchById(self.GuestId)
	if guest != nil {
		return guest.(*SGuest)
	}
	return nil
}

func (gn *SGuestnetwork) GetNetwork() *SNetwork {
	net, _ := NetworkManager.FetchById(gn.NetworkId)
	if net != nil {
		return net.(*SNetwork)
	}
	return nil
}

func (self *SGuestnetwork) GetTeamGuestnetwork() (*SGuestnetwork, error) {
	if len(self.TeamWith) > 0 {
		return GuestnetworkManager.FetchByIdsAndIpMac(self.GuestId, self.NetworkId, "", self.TeamWith)
	}
	return nil, nil
}

func (self *SGuestnetwork) getJsonDescAtBaremetal(host *SHost) jsonutils.JSONObject {
	network := self.GetNetwork()
	hostwire := host.getHostwireOfIdAndMac(network.WireId, self.MacAddr)
	return self.getGeneralJsonDesc(host, network, hostwire)
}

func guestGetHostWireFromNetwork(host *SHost, network *SNetwork) (*SHostwire, error) {
	hostwires := host.getHostwiresOfId(network.WireId)
	var hostWire *SHostwire
	for i := 0; i < len(hostwires); i++ {
		if netInter, _ := NetInterfaceManager.FetchByMac(hostwires[i].MacAddr); netInter != nil {
			if netInter.NicType != api.NIC_TYPE_IPMI {
				hostWire = &hostwires[i]
				break
			}
		}
	}
	if hostWire == nil {
		return nil, fmt.Errorf("Host %s has no net interface on wire %s as guest network %s",
			host.Name, network.WireId, api.NIC_TYPE_ADMIN)
	}
	return hostWire, nil
}

func (self *SGuestnetwork) getJsonDescAtHost(host *SHost) jsonutils.JSONObject {
	network := self.GetNetwork()
	hostWire, err := guestGetHostWireFromNetwork(host, network)
	if err != nil {
		log.Errorln(err)
	}
	return self.getGeneralJsonDesc(host, network, hostWire)
}

func (self *SGuestnetwork) getGeneralJsonDesc(host *SHost, network *SNetwork, hostwire *SHostwire) jsonutils.JSONObject {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(network.Name), "net")
	desc.Add(jsonutils.NewString(self.NetworkId), "net_id")
	desc.Add(jsonutils.NewString(self.MacAddr), "mac")
	if self.Virtual {
		desc.Add(jsonutils.JSONTrue, "virtual")
		if len(self.TeamWith) > 0 {
			teamGN, _ := self.GetTeamGuestnetwork()
			if teamGN != nil {
				log.Debugf("%#v", teamGN)
				desc.Add(jsonutils.NewString(teamGN.IpAddr), "ip")
			}
		} else {
			desc.Add(jsonutils.NewString(network.GetNetAddr().String()), "ip")
		}
	} else {
		desc.Add(jsonutils.JSONFalse, "virtual")
		desc.Add(jsonutils.NewString(self.IpAddr), "ip")
	}
	if len(network.GuestGateway) > 0 {
		desc.Add(jsonutils.NewString(network.GuestGateway), "gateway")
	}
	desc.Add(jsonutils.NewString(network.GetDNS()), "dns")
	desc.Add(jsonutils.NewString(network.GetDomain()), "domain")
	routes := network.GetRoutes()
	if routes != nil && len(routes) > 0 {
		desc.Add(jsonutils.Marshal(routes), "routes")
	}
	desc.Add(jsonutils.NewString(self.GetIfname()), "ifname")
	desc.Add(jsonutils.NewInt(int64(network.GuestIpMask)), "masklen")
	desc.Add(jsonutils.NewString(self.Driver), "driver")
	desc.Add(jsonutils.NewString(hostwire.Bridge), "bridge")
	desc.Add(jsonutils.NewString(hostwire.WireId), "wire_id")
	desc.Add(jsonutils.NewInt(int64(network.VlanId)), "vlan")
	desc.Add(jsonutils.NewString(hostwire.Interface), "interface")
	desc.Add(jsonutils.NewInt(int64(self.getBandwidth())), "bw")
	desc.Add(jsonutils.NewInt(int64(self.Index)), "index")
	vips := self.GetVirtualIPs()
	if len(vips) > 0 {
		desc.Add(jsonutils.NewStringArray(vips), "virtual_ips")
	}
	if len(network.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(network.ExternalId), "external_id")
	}

	if len(self.TeamWith) > 0 {
		desc.Add(jsonutils.NewString(self.TeamWith), "team_with")
	}

	guest := self.getGuest()
	if guest.GetHypervisor() != api.HYPERVISOR_KVM {
		desc.Add(jsonutils.JSONTrue, "manual")
	}

	return desc
}

func (manager *SGuestnetworkManager) GetGuestByAddress(address string) *SGuest {
	networks := manager.TableSpec().Instance()
	guests := GuestManager.Query()
	q := guests.Join(networks, sqlchemy.AND(
		sqlchemy.IsFalse(networks.Field("deleted")),
		sqlchemy.Equals(networks.Field("ip_addr"), address),
		sqlchemy.Equals(networks.Field("guest_id"), guests.Field("id")),
	))
	guest := &SGuest{}
	guest.SetModelManager(GuestManager, guest)
	err := q.First(guest)
	if err == nil {
		return guest
	}
	return nil
}

func (self *SGuestnetwork) GetDetailedString() string {
	network := self.GetNetwork()
	return fmt.Sprintf("eth%d:%s/%d/%s/%d/%s/%s/%d", self.Index, self.IpAddr, network.GuestIpMask,
		self.MacAddr, network.VlanId, network.Name, self.Driver, self.getBandwidth())
}

func (self *SGuestnetwork) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("index") {
		index, err := data.Int("index")
		if err != nil {
			return nil, httperrors.NewInternalServerError("fail to fetch index %s", err)
		}
		q := GuestnetworkManager.Query().SubQuery()
		count, err := q.Query().Filter(sqlchemy.Equals(q.Field("guest_id"), self.GuestId)).
			Filter(sqlchemy.NotEquals(q.Field("network_id"), self.NetworkId)).
			Filter(sqlchemy.Equals(q.Field("index"), index)).CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("checkout nic index uniqueness fail %s", err)
		}
		if count > 0 {
			return nil, httperrors.NewDuplicateResourceError("NIC Index %d has been occupied", index)
		}
	}
	return self.SJointResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGuestnetworkManager) DeleteGuestNics(ctx context.Context, userCred mcclient.TokenCredential, gns []SGuestnetwork, reserve bool) error {
	for i := range gns {
		gn := gns[i]
		guest := gn.GetGuest()
		net := gn.GetNetwork()
		if regutils.MatchIP4Addr(gn.IpAddr) || regutils.MatchIP6Addr(gn.Ip6Addr) {
			net.updateDnsRecord(&gn, false)
			if regutils.MatchIP4Addr(gn.IpAddr) {
				// ??
				// netman.get_manager().netmap_remove_node(gn.ip_addr)
			}
		}
		// ??
		// gn.Delete(ctx, userCred)
		err := gn.Delete(ctx, userCred)
		if err != nil {
			log.Errorf("%s", err)
		}
		gn.LogDetachEvent(ctx, userCred, guest, net)
		if reserve && regutils.MatchIP4Addr(gn.IpAddr) {
			ReservedipManager.ReserveIP(userCred, net, gn.IpAddr, "Delete to reserve")
		}
	}
	return nil
}

func (manager *SGuestnetworkManager) getGuestNicByIP(ip string, networkId string) (*SGuestnetwork, error) {
	gn := SGuestnetwork{}
	q := manager.Query()
	q = q.Equals("ip_addr", ip).Equals("network_id", networkId)
	err := q.First(&gn)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("getGuestNicByIP fail %s", err)
			return nil, err
		}
		return nil, nil
	}
	gn.SetModelManager(manager, &gn)
	return &gn, nil
}

func (self *SGuestnetwork) LogDetachEvent(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, network *SNetwork) {
	if network == nil {
		netTmp, _ := NetworkManager.FetchById(self.NetworkId)
		network = netTmp.(*SNetwork)
	}
	db.OpsLog.LogDetachEvent(ctx, guest, network, userCred, nil)
}

func (self *SGuestnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func totalGuestNicCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, includeSystem bool) GuestnicsCount {
	guests := GuestManager.Query().SubQuery()
	guestnics := GuestnetworkManager.Query().SubQuery()
	q := guestnics.Query().Join(guests, sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")))
	if rangeObj != nil {
		if rangeObj.Keyword() == "zone" {
			hosts := HostManager.Query().SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id"))).Filter(sqlchemy.Equals(hosts.Field("zone_id"), rangeObj.GetId()))
		} else if rangeObj.Keyword() == "wire" {
			hosts := HostManager.Query().SubQuery()
			hostwires := HostwireManager.Query().SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id"))).Join(hostwires, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id"))).Filter(sqlchemy.Equals(hostwires.Field("wire_id"), rangeObj.GetId()))
		} else if rangeObj.Keyword() == "host" {

		} else if rangeObj.Keyword() == "vcenter" {

		} else if rangeObj.Keyword() == "schedtag" {

		}
	}

	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(guests.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(guests.Field("tenant_id"), ownerId.GetProjectId()))
	}

	if !includeSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(guests.Field("is_system")),
			sqlchemy.IsFalse(guests.Field("is_system"))))
	}
	return calculateNics(q)
}

type GuestnicsCount struct {
	InternalNicCount        int
	InternalVirtualNicCount int
	ExternalNicCount        int
	ExternalVirtualNicCount int
	InternalBandwidth       int
	ExternalBandwidth       int
}

func calculateNics(q *sqlchemy.SQuery) GuestnicsCount {
	cnt := GuestnicsCount{}
	gns := make([]SGuestnetwork, 0)
	err := db.FetchModelObjects(GuestnetworkManager, q, &gns)
	if err != nil {
		log.Errorf("guestnics total count query error %s", err)
	}
	for _, gn := range gns {
		if gn.IsExit() {
			if gn.Virtual {
				cnt.ExternalVirtualNicCount += 1
			} else {
				cnt.ExternalNicCount += 1
			}
			cnt.ExternalBandwidth += gn.BwLimit
		} else {
			if gn.Virtual {
				cnt.InternalVirtualNicCount += 1
			} else {
				cnt.InternalNicCount += 1
			}
			cnt.InternalBandwidth += gn.BwLimit
		}
	}
	return cnt
}

func (self *SGuestnetwork) IsExit() bool {
	net := self.GetNetwork()
	if net != nil {
		return net.IsExitNetwork()
	}
	return false
}

func (self *SGuestnetwork) getBandwidth() int {
	if self.BwLimit > 0 && self.BwLimit <= api.MAX_BANDWIDTH {
		return self.BwLimit
	} else {
		net := self.GetNetwork()
		if net != nil {
			wire := net.GetWire()
			if wire != nil {
				return wire.Bandwidth
			}
		}
		return options.Options.DefaultBandwidth
	}
}

func (self *SGuestnetwork) IsAllocated() bool {
	if regutils.MatchMacAddr(self.MacAddr) && (self.Virtual || regutils.MatchIP4Addr(self.IpAddr)) {
		return true
	} else {
		return false
	}
}

func GetIPTenantIdPairs() {
	/*
			from guests import Guests
		        from hosts import Hosts
		        from sqlalchemy.sql.expression import bindparam
		        q = Guestnics.query(Guestnics.ip_addr, Guestnics.mac_addr,
		                            bindparam('tunnel_key', None), # XXX: tunnel_key for VPC
		                            Guests.tenant_id, Guests.name,
		                            bindparam('tunnel_ip', None), Hosts.access_ip) \
		                        .join(Guests, and_(Guests.id==Guestnics.guest_id,
		                                            Guests.deleted==False)) \
		                        .join(Hosts, and_(Hosts.id==Guests.host_id,
		                                            Hosts.deleted==False))
		        return q.all()
	*/
}

func (self *SGuestnetwork) GetVirtualIPs() []string {
	ips := make([]string, 0)
	guest := self.GetGuest()
	net := self.GetNetwork()
	for _, guestgroup := range guest.GetGroups() {
		group := guestgroup.GetGroup()
		groupnets, err := group.GetNetworks()
		if err != nil {
			continue
		}
		for _, groupnetwork := range groupnets {
			gnet := groupnetwork.GetNetwork()
			if gnet.WireId == net.WireId {
				ips = append(ips, groupnetwork.IpAddr)
			}
		}
	}
	return ips
}

func (self *SGuestnetwork) GetIfname() string {
	return self.Ifname
}

func (manager *SGuestnetworkManager) getRecentlyReleasedIPAddresses(networkId string, recentDuration time.Duration) map[string]bool {
	if recentDuration == 0 {
		return nil
	}
	since := time.Now().UTC().Add(-recentDuration)
	q := manager.RawQuery("ip_addr")
	q = q.Equals("network_id", networkId).IsTrue("deleted")
	q = q.GT("deleted_at", since).Distinct()
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("GetRecentlyReleasedIPAddresses fail %s", err)
		return nil
	}
	defer rows.Close()
	ret := make(map[string]bool)
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			log.Errorf("scan error %s", err)
		} else {
			ret[ip] = true
		}
	}
	return ret
}

func (manager *SGuestnetworkManager) FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery {
	macStr := jsonutils.GetAnyString(params, []string{"mac", "mac_addr"})
	if len(macStr) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("mac_addr"), macStr))
	}
	ipStr := jsonutils.GetAnyString(params, []string{"ipaddr", "ip_addr", "ip"})
	if len(ipStr) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("ip_addr"), ipStr))
	}
	ip6Str := jsonutils.GetAnyString(params, []string{"ip6addr", "ip6_addr", "ip6"})
	if len(ip6Str) > 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("ip6_addr"), ip6Str))
	}
	return q
}

func (manager *SGuestnetworkManager) FetchByIdsAndIpMac(guestId string, netId string, ipAddr string, mac string) (*SGuestnetwork, error) {
	query := jsonutils.NewDict()
	if len(mac) > 0 {
		query.Add(jsonutils.NewString(mac), "mac_addr")
	}
	if len(ipAddr) > 0 {
		query.Add(jsonutils.NewString(ipAddr), "ip_addr")
	}
	ign, err := db.FetchJointByIds(manager, guestId, netId, query)
	if err != nil {
		return nil, err
	}
	return ign.(*SGuestnetwork), nil
}

func (self *SGuestnetwork) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	if len(self.IpAddr) > 0 {
		desc.Add(jsonutils.NewString(self.IpAddr), "ip_addr")
	}
	if len(self.Ip6Addr) > 0 {
		desc.Add(jsonutils.NewString(self.Ip6Addr), "ip6_addr")
	}
	desc.Add(jsonutils.NewString(self.MacAddr), "mac")
	if len(self.TeamWith) > 0 {
		desc.Add(jsonutils.NewString(self.TeamWith), "team_with")
	}
	return desc
}

func (self *SGuestnetwork) ToNetworkConfig() *api.NetworkConfig {
	net := self.GetNetwork()
	if net == nil {
		return nil
	}
	ret := &api.NetworkConfig{
		Index:   int(self.Index),
		Network: net.Id,
		Wire:    net.GetWire().Id,
		Mac:     self.MacAddr,
		Address: self.IpAddr,
		Driver:  self.Driver,
		BwLimit: self.BwLimit,
		Project: net.ProjectId,
		Domain:  net.DomainId,
		Ifname:  self.Ifname,
		NetType: net.ServerType,
	}
	return ret
}
