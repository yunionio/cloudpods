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
	"math/rand"
	"regexp"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	randutil "yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	MAX_IFNAME_SIZE = 13
	MAX_HINT_LEN    = MAX_IFNAME_SIZE - 4          // 9
	HINT_BASE_LEN   = 6                            // 6
	HINT_RAND_LEN   = MAX_HINT_LEN - HINT_BASE_LEN // 3

	MAX_GUESTNIC_TO_SAME_NETWORK = 2
)

type SGuestnetworkManager struct {
	SGuestJointsManager
	SNetworkResourceBaseManager
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

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" `

	// MAC地址
	MacAddr string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	// IPv4地址
	IpAddr string `width:"16" charset:"ascii" nullable:"false" list:"user"`
	// IPv6地址
	Ip6Addr string `width:"64" charset:"ascii" nullable:"true" list:"user"`
	// 虚拟网卡驱动
	Driver string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"`
	// 带宽限制，单位mbps
	BwLimit int `nullable:"false" default:"0" list:"user"`
	// 网卡序号
	Index int8 `nullable:"false" default:"0" list:"user" update:"user"`
	// 是否为虚拟接口（无IP）
	Virtual bool `default:"false" list:"user"`
	// 虚拟网卡设备名称
	Ifname string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"`

	// bind配对网卡MAC地址
	TeamWith string `width:"32" charset:"ascii" nullable:"false" list:"user"`

	// IPv4映射地址，当子网属于私有云vpc的时候分配，用于访问外网
	MappedIpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user"`

	// 网卡关联的Eip实例
	EipId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
}

func (manager *SGuestnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (self *SGuestnetwork) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.GuestnetworkDetails, error) {
	return api.GuestnetworkDetails{}, nil
}

func (manager *SGuestnetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestnetworkDetails {
	rows := make([]api.GuestnetworkDetails, len(objs))

	guestRows := manager.SGuestJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GuestnetworkDetails{
			GuestJointResourceDetails: guestRows[i],
		}
		netIds[i] = objs[i].(*SGuestnetwork).NetworkId
		iNet, _ := NetworkManager.FetchById(netIds[i])
		net := iNet.(*SNetwork)
		rows[i].WireId = net.WireId
	}

	netIdMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := netIdMaps[netIds[i]]; ok {
			rows[i].Network = name
		}
	}

	return rows
}

func (manager *SGuestnetworkManager) fetchByRowIds(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	rowIds []int64,
) ([]SGuestnetwork, error) {
	var gns []SGuestnetwork
	q := manager.Query().In("row_id", rowIds)
	if err := db.FetchModelObjects(manager, q, &gns); err != nil {
		return nil, errors.Wrapf(err, "fetch guestnetworks by row_id list")
	}
	idmap := map[int64]int{}
	for i := range gns {
		idmap[gns[i].RowId] = i
	}
	ret := make([]SGuestnetwork, len(rowIds))
	for i, rowId := range rowIds {
		if j, ok := idmap[rowId]; ok {
			ret[i] = gns[j]
		} else {
			return nil, errors.Wrapf(errors.ErrNotFound, "guestnetwork row %d", rowId)
		}
	}
	return ret, nil
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

type newGuestNetworkArgs struct {
	guest   *SGuest
	network *SNetwork

	index int8

	ipAddr              string
	allocDir            api.IPAllocationDirection
	tryReserved         bool
	requireDesignatedIP bool
	useDesignatedIP     bool

	ifname      string
	macAddr     string
	bwLimit     int
	nicDriver   string
	teamWithMac string

	virtual bool
}

func (manager *SGuestnetworkManager) newGuestNetwork(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	args newGuestNetworkArgs,
) (*SGuestnetwork, error) {
	gn := SGuestnetwork{}
	gn.SetModelManager(GuestnetworkManager, &gn)

	var (
		guest                = args.guest
		network              = args.network
		index                = args.index
		address              = args.ipAddr
		mac                  = args.macAddr
		driver               = args.nicDriver
		bwLimit              = args.bwLimit
		virtual              = args.virtual
		reserved             = args.tryReserved
		allocDir             = args.allocDir
		requiredDesignatedIp = args.requireDesignatedIP
		reUseAddr            = args.useDesignatedIP
		ifname               = args.ifname
		teamWithMac          = args.teamWithMac
	)

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
		if len(address) > 0 && reUseAddr {
			ipAddr, err := netutils.NewIPV4Addr(address)
			if err != nil {
				return nil, errors.Wrapf(err, "Reuse invalid address %s", address)
			}
			if !network.IsAddressInRange(ipAddr) {
				return nil, errors.Wrapf(httperrors.ErrOutOfRange, "%s not in network address range", address)
			}
			// if reuse Ip address, no need to check address availability
			// assign it anyway
			gn.IpAddr = address
		} else {
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

		if vpc := network.GetVpc(); vpc == nil {
			return nil, fmt.Errorf("cannot find vpc of network %s(%s)", network.Id, network.Name)
		} else if vpc.Id != api.DEFAULT_VPC_ID && vpc.GetProviderName() == api.CLOUD_PROVIDER_ONECLOUD {
			var err error
			GuestnetworkManager.lockAllocMappedAddr(ctx)
			defer GuestnetworkManager.unlockAllocMappedAddr(ctx)
			gn.MappedIpAddr, err = GuestnetworkManager.allocMappedIpAddr(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	ifname, err = gn.checkOrAllocateIfname(network, ifname)
	if err != nil {
		return nil, err
	}
	gn.Ifname = ifname
	gn.TeamWith = teamWithMac
	err = manager.TableSpec().Insert(ctx, &gn)
	if err != nil {
		return nil, err
	}
	return &gn, nil
}

func (self *SGuestnetwork) generateIfname(network *SNetwork, virtual bool, randomized bool) string {
	// It may happen that external networks when synced can miss ifname hint
	network.ensureIfnameHint()

	pattern := regexp.MustCompile(`\W+`)
	nName := pattern.ReplaceAllString(network.IfnameHint, "")
	if len(nName) > MAX_IFNAME_SIZE-4 {
		nName = nName[:(MAX_IFNAME_SIZE - 4)]
	}
	if virtual {
		return fmt.Sprintf("%s-%s", nName, randutil.String(3))
	} else {
		ip, _ := netutils.NewIPV4Addr(self.IpAddr)
		cliaddr := ip.CliAddr(network.GuestIpMask)
		return fmt.Sprintf("%s-%d", nName, uint32(cliaddr))
	}
}

func (man *SGuestnetworkManager) ifnameUsed(ifname string) bool {
	// inviable names are always used
	if ifname == "" {
		return true
	}
	if len(ifname) > MAX_IFNAME_SIZE {
		return true
	}
	isa := func(c byte) bool {
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	if !isa(ifname[0]) {
		return true
	}
	for i := range ifname[1:] {
		c := ifname[i]
		if isa(c) || c >= '0' || c <= '9' || c == '_' || c == '-' {
			continue
		}
		return true
	}

	count, err := GuestnetworkManager.Query().Equals("ifname", ifname).CountWithError()
	if err != nil {
		panic(errors.Wrap(err, "query if ifname is used"))
	}
	return count > 0
}

func (self *SGuestnetwork) checkOrAllocateIfname(network *SNetwork, preferIfname string) (string, error) {
	man := GuestnetworkManager
	if !man.ifnameUsed(preferIfname) {
		return preferIfname, nil
	}
	ifname := self.generateIfname(network, self.Virtual, false)
	if !man.ifnameUsed(ifname) {
		return ifname, nil
	}
	if !self.Virtual {
		ifname = self.generateIfname(network, true, false)
	}
	found := false
	for i := 0; i < 5; i++ {
		if !man.ifnameUsed(ifname) {
			found = true
			break
		}
		ifname = self.generateIfname(network, true, true)
	}
	if !found {
		return "", httperrors.NewConflictError("cannot allocate ifname")
	}
	return ifname, nil
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
	return self.getJsonDescHostwire(network, hostwire)
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
	if network.isOneCloudVpcNetwork() {
		return self.getJsonDescOneCloudVpc(network)
	} else {
		hostWire, err := guestGetHostWireFromNetwork(host, network)
		if err != nil {
			log.Errorln(err)
		}
		return self.getJsonDescHostwire(network, hostWire)
	}
}

func (self *SGuestnetwork) getJsonDescHostwire(network *SNetwork, hostwire *SHostwire) *jsonutils.JSONDict {
	desc := self.getJsonDesc(network)
	if hostwire != nil {
		desc.Add(jsonutils.NewString(hostwire.Bridge), "bridge")
		desc.Add(jsonutils.NewString(hostwire.WireId), "wire_id")
		desc.Add(jsonutils.NewString(hostwire.Interface), "interface")
	}
	return desc
}

func (self *SGuestnetwork) getJsonDescOneCloudVpc(network *SNetwork) *jsonutils.JSONDict {
	if self.MappedIpAddr == "" {
		var (
			err  error
			addr string
		)
		addr, err = GuestnetworkManager.allocMappedIpAddr(context.TODO())
		if err != nil {
			log.Errorf("getJsonDescOneCloudVpc: row %d: alloc mapped ipaddr: %v", self.RowId, err)
		} else {
			if _, err := db.Update(self, func() error {
				self.MappedIpAddr = addr
				return nil
			}); err != nil {
				log.Errorf("getJsonDescOneCloudVpc: row %d: db update mapped addr: %v", self.RowId, err)
				self.MappedIpAddr = ""
			}
		}
	}
	vpc := network.GetVpc()

	vpcDesc := jsonutils.NewDict()
	vpcDesc.Set("id", jsonutils.NewString(vpc.Id))
	vpcDesc.Set("provider", jsonutils.NewString(api.VPC_PROVIDER_OVN))
	vpcDesc.Set("mapped_ip_addr", jsonutils.NewString(self.MappedIpAddr))
	desc := self.getJsonDesc(network)
	desc.Set("vpc", vpcDesc)
	return desc
}

func (self *SGuestnetwork) getJsonDesc(network *SNetwork) *jsonutils.JSONDict {
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
	desc.Add(jsonutils.NewInt(int64(network.VlanId)), "vlan")
	desc.Add(jsonutils.NewInt(int64(self.getBandwidth())), "bw")
	desc.Add(jsonutils.NewInt(int64(self.getMtu())), "mtu")
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

func (self *SGuestnetwork) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestnetworkUpdateInput,
) (api.GuestnetworkUpdateInput, error) {
	if input.Index != nil {
		index := *input.Index
		q := GuestnetworkManager.Query().SubQuery()
		count, err := q.Query().Filter(sqlchemy.Equals(q.Field("guest_id"), self.GuestId)).
			Filter(sqlchemy.NotEquals(q.Field("network_id"), self.NetworkId)).
			Filter(sqlchemy.Equals(q.Field("index"), index)).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("checkout nic index uniqueness fail %s", err)
		}
		if count > 0 {
			return input, httperrors.NewDuplicateResourceError("NIC Index %d has been occupied", index)
		}
	}
	var err error
	input.GuestJointBaseUpdateInput, err = self.SGuestJointsBase.ValidateUpdateData(ctx, userCred, query, input.GuestJointBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SGuestJointsBase.ValidateUpdateData")
	}
	return input, nil
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
	if err := NetworkAddressManager.deleteByGuestnetworkId(ctx, userCred, self.RowId); err != nil {
		return errors.Wrap(err, "delete attached network addresses")
	}
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func totalGuestNicCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	includeSystem bool,
	providers []string,
	brands []string,
	cloudEnv string,
) GuestnicsCount {
	guests := GuestManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()
	guestnics := GuestnetworkManager.Query().SubQuery()

	q := guestnics.Query()
	q = q.Join(guests, sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")))
	q = q.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))

	q = q.Filter(sqlchemy.IsFalse(guests.Field("pending_deleted")))

	q = CloudProviderFilter(q, hosts.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, nil, hosts.Field("zone_id"), hosts.Field("manager_id"), hosts.Field("id"), nil)

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
	if self.IpAddr != "" {
		addr, err := netutils.NewIPV4Addr(self.IpAddr)
		if err == nil {
			return netutils.IsExitAddress(addr)
		}
	}
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

func (self *SGuestnetwork) getMtu() int {
	net := self.GetNetwork()
	if net != nil {
		wire := net.GetWire()
		if wire != nil {
			return wire.Mtu
		}
	}
	return options.Options.DefaultMtu
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

func (manager *SGuestnetworkManager) FetchByGuestIdIndex(guestId string, index int8) (*SGuestnetwork, error) {
	q := manager.Query().
		Equals("guest_id", guestId).
		Equals("index", index)
	var rets []SGuestnetwork
	if err := db.FetchModelObjects(manager, q, &rets); err != nil {
		return nil, err
	}
	if len(rets) > 1 {
		return nil, errors.Errorf("guest %s has conflict nic index (%d)", guestId, index)
	}
	if len(rets) == 0 {
		return nil, errors.ErrNotFound
	}
	return &rets[0], nil
}

func (self *SGuestnetwork) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := api.GuestnetworkShortDesc{}
	if len(self.IpAddr) > 0 {
		desc.IpAddr = self.IpAddr
		desc.IsExit = self.IsExit()
	}
	if len(self.Ip6Addr) > 0 {
		desc.Ip6Addr = self.Ip6Addr
	}
	desc.Mac = self.MacAddr
	if len(self.TeamWith) > 0 {
		desc.TeamWith = self.TeamWith
	}
	return jsonutils.Marshal(desc).(*jsonutils.JSONDict)
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

func (manager *SGuestnetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(query.MacAddr) > 0 {
		q = q.In("mac_addr", query.MacAddr)
	}
	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}
	if len(query.Ip6Addr) > 0 {
		q = q.In("ip6_addr", query.Ip6Addr)
	}
	if len(query.Driver) > 0 {
		q = q.In("driver", query.Driver)
	}
	if len(query.Ifname) > 0 {
		q = q.In("ifname", query.Ifname)
	}
	if len(query.TeamWith) > 0 {
		q = q.In("team_with", query.TeamWith)
	}

	return q, nil
}

func (manager *SGuestnetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.OrderByExtraFields(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGuestnetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SNetworkResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
