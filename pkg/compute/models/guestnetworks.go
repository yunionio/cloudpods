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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
)

const (
	MAX_IFNAME_SIZE = 13
)

type SGuestnetworkManager struct {
	SGuestJointsManager
}

var GuestnetworkManager *SGuestnetworkManager

func init() {
	db.InitManager(func() {
		GuestnetworkManager = &SGuestnetworkManager{SGuestJointsManager: NewGuestJointsManager(SGuestnetwork{},
			"guestnetworks_tbl", "guestnetwork", "guestnetworks", NetworkManager)}
	})
}

type SGuestnetwork struct {
	SGuestJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user"  key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	MacAddr   string `width:"32" charset:"ascii" nullable:"false" list:"user"`                   // Column(VARCHAR(32, charset='ascii'), nullable=False)
	IpAddr    string `width:"16" charset:"ascii" nullable:"false" list:"user"`                   // Column(VARCHAR(16, charset='ascii'), nullable=True)
	Ip6Addr   string `width:"64" charset:"ascii" nullable:"true" list:"user"`                    // Column(VARCHAR(64, charset='ascii'), nullable=True)
	Driver    string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"`      // Column(VARCHAR(16, charset='ascii'), nullable=True)
	BwLimit   int    `nullable:"false" default:"0" list:"user"`                                  // Column(Integer, nullable=False, default=0) # Mbps
	Index     int8   `nullable:"false" default:"0" list:"user" update:"user"`                    // Column(TINYINT, nullable=False, default=0)
	Virtual   bool   `default:"false" list:"user"`                                               // Column(Boolean, default=False)
	Ifname    string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"`      // Column(VARCHAR(16, charset='ascii'), nullable=True)
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

func (self *SGuestnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGuestJointsBase.GetExtraDetails(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
}

func (manager *SGuestnetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (gn *SGuestnetwork) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

const MAX_TRIES = 10

func (manager *SGuestnetworkManager) GenerateMac(netId string, suggestion string) string {
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
		if q.Count() == 0 {
			return mac
		}
	}
	return ""
}

func (manager *SGuestnetworkManager) newGuestNetwork(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, network *SNetwork,
	index int8, address string, mac string, driver string, bwLimit int, virtual bool, reserved bool,
	allocDir IPAddlocationDirection, requiredDesignatedIp bool) (*SGuestnetwork, error) {

	gn := SGuestnetwork{}
	gn.SetModelManager(GuestnetworkManager)

	gn.GuestId = guest.Id
	gn.NetworkId = network.Id
	gn.Index = index
	gn.Virtual = virtual
	if len(driver) == 0 {
		driver = "virtio"
	}
	gn.Driver = driver
	if bwLimit > 0 {
		gn.BwLimit = bwLimit
	}

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)

	macAddr := manager.GenerateMac(network.Id, mac)
	if len(macAddr) == 0 {
		log.Errorf("Mac address generate fails")
		return nil, fmt.Errorf("mac address generate fails")
	}
	gn.MacAddr = macAddr
	if !virtual {
		addrTable := network.GetUsedAddresses()
		recentAddrTable := manager.getRecentlyReleasedIPAddresses(network.Id, time.Duration(network.AllocTimoutSeconds)*time.Second)
		ipAddr, err := network.GetFreeIP(ctx, userCred, addrTable, recentAddrTable, address, allocDir, reserved)
		if err != nil {
			return nil, err
		}
		if len(address) > 0 && ipAddr != address && requiredDesignatedIp {
			return nil, fmt.Errorf("candidate ip %s is occupoed!", address)
		}
		gn.IpAddr = ipAddr
	}
	ifTable := network.GetUsedIfnames()
	ifName := gn.GetFreeIfname(network, ifTable)
	gn.Ifname = ifName
	err := manager.TableSpec().Insert(&gn)
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
		io.WriteString(hash, fmt.Sprintf("%d", time.Now().String()))
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

func (self *SGuestnetwork) GetJsonDescAtHost(host *SHost) jsonutils.JSONObject {
	network := self.GetNetwork()
	hostwire := host.getHostwireOfId(network.WireId)

	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(network.Name), "net")
	desc.Add(jsonutils.NewString(self.NetworkId), "net_id")
	desc.Add(jsonutils.NewString(self.MacAddr), "mac")
	if self.Virtual {
		desc.Add(jsonutils.JSONTrue, "virtual")
		desc.Add(jsonutils.NewString(network.GetNetAddr().String()), "ip")
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
	if len(routes) > 0 {
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
	guest := self.GetGuest()
	if guest.GetHypervisor() != HYPERVISOR_KVM {
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
	guest.SetModelManager(GuestManager)
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
			return nil, fmt.Errorf("fail to fetch index %s", err)
		}
		q := GuestnetworkManager.Query().SubQuery()
		count := q.Query().Filter(sqlchemy.Equals(q.Field("guest_id"), self.GuestId)).
			Filter(sqlchemy.NotEquals(q.Field("network_id"), self.NetworkId)).
			Filter(sqlchemy.Equals(q.Field("index"), index)).Count()
		if count > 0 {
			return nil, fmt.Errorf("NIC Index %d has been occupied", index)
		}
	}
	return self.SJointResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGuestnetworkManager) DeleteGuestNics(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential, network *SNetwork, reserve bool) error {
	q := manager.Query().Equals("guest_id", guest.Id)
	if network != nil {
		q = q.Equals("network_id", network.Id)
	}
	gns := make([]SGuestnetwork, 0)
	err := db.FetchModelObjects(manager, q, &gns)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	for _, gn := range gns {
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
		err = gn.Delete(ctx, userCred)
		if err != nil {
			log.Errorf("%s", err)
		}
		gn.LogDetachEvent(userCred, guest, net)
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
	gn.SetModelManager(manager)
	return &gn, nil
}

func (self *SGuestnetwork) LogDetachEvent(userCred mcclient.TokenCredential, guest *SGuest, network *SNetwork) {
	if network == nil {
		netTmp, _ := NetworkManager.FetchById(self.NetworkId)
		network = netTmp.(*SNetwork)
	}
	db.OpsLog.LogDetachEvent(guest, network, userCred, nil)
}

func (self *SGuestnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func totalGuestNicCount(projectId string, rangeObj db.IStandaloneModel, includeSystem bool) GuestnicsCount {
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
	if len(projectId) > 0 {
		q = q.Filter(sqlchemy.Equals(guests.Field("tenant_id"), projectId))
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
	if self.BwLimit > 0 && self.BwLimit <= MAX_BANDWIDTH {
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
		for _, groupnetwork := range group.GetNetworks() {
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

func (self *SGuestnetwork) getJsonDescAtHost(host *SHost) jsonutils.JSONObject {
	desc := jsonutils.NewDict()

	network := self.GetNetwork()
	hostwire := host.getHostwireOfId(network.WireId)

	desc.Add(jsonutils.NewString(network.Name), "net")
	desc.Add(jsonutils.NewString(self.NetworkId), "net_id")
	desc.Add(jsonutils.NewString(self.MacAddr), "mac")
	if self.Virtual {
		desc.Add(jsonutils.JSONTrue, "virtual")
		desc.Add(jsonutils.NewString(network.GetNetAddr().String()), "ip")
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

	guest := self.getGuest()
	if guest.GetHypervisor() != HYPERVISOR_KVM {
		desc.Add(jsonutils.JSONTrue, "manual")
	}

	return desc
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
