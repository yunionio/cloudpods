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
	"regexp"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	randutil "yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
		GuestnetworkManager.TableSpec().AddIndex(false, "mac_addr", "deleted")
	})
}

type SGuestnetwork struct {
	SGuestJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" `

	// MAC地址
	MacAddr string `width:"32" charset:"ascii" nullable:"false" list:"user" index:"true"`
	// IPv4地址
	IpAddr string `width:"16" charset:"ascii" nullable:"false" list:"user"`
	// IPv6地址
	Ip6Addr string `width:"64" charset:"ascii" nullable:"true" list:"user"`
	// 虚拟网卡驱动
	Driver string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user"`
	// 网卡队列数
	NumQueues int `nullable:"true" default:"1" list:"user" update:"user"`
	// 带宽限制，单位mbps
	BwLimit int `nullable:"false" default:"0" list:"user"`
	// 下行流量限制，单位 bytes
	RxTrafficLimit int64 `nullable:"false" default:"0" list:"user"`
	RxTrafficUsed  int64 `nullable:"false" default:"0" list:"user"`
	// 上行流量限制，单位 bytes
	TxTrafficLimit int64 `nullable:"false" default:"0" list:"user"`
	TxTrafficUsed  int64 `nullable:"false" default:"0" list:"user"`
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

func (gn SGuestnetwork) GetIP() string {
	return gn.IpAddr
}

func (gn SGuestnetwork) GetMAC() string {
	return gn.MacAddr
}

func (manager *SGuestnetworkManager) GetSlaveFieldName() string {
	return "network_id"
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
	eipIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GuestnetworkDetails{
			GuestJointResourceDetails: guestRows[i],
		}
		netIds[i] = objs[i].(*SGuestnetwork).NetworkId
		ipnets, err := NetworkAddressManager.fetchAddressesByGuestnetworkId(objs[i].(*SGuestnetwork).RowId)
		if err != nil {
			log.Errorln(err)
		} else if len(ipnets) > 0 {
			rows[i].NetworkAddresses = ipnets
		}
		iNet, _ := NetworkManager.FetchById(netIds[i])
		net := iNet.(*SNetwork)
		rows[i].WireId = net.WireId
		eipIds[i] = objs[i].(*SGuestnetwork).EipId
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

	eipIdMaps, err := db.FetchIdFieldMap2(ElasticipManager, "ip_addr", eipIds)
	if err != nil {
		log.Errorf("FetchIdFieldMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if ip, ok := eipIdMaps[eipIds[i]]; ok {
			rows[i].EipAddr = ip
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

func (manager *SGuestnetworkManager) fetchByRowId(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	rowId int64,
) (*SGuestnetwork, error) {
	gns, err := manager.fetchByRowIds(ctx, userCred, []int64{rowId})
	if err != nil {
		return nil, err
	}
	if len(gns) != 1 {
		return nil, errors.Errorf("row_id %d: got %d guesnetwork entries", rowId, len(gns))
	}
	return &gns[0], nil
}

func (manager *SGuestnetworkManager) GenerateMac(suggestion string) (string, error) {
	return generateMac(suggestion)
}

func (manager *SGuestnetworkManager) FilterByMac(mac string) *sqlchemy.SQuery {
	return manager.Query().Equals("mac_addr", mac)
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

	ifname         string
	macAddr        string
	bwLimit        int
	nicDriver      string
	numQueues      int
	teamWithMac    string
	rxTrafficLimit int64
	txTrafficLimit int64

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
		numQueues            = args.numQueues
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
		driver = api.NETWORK_DRIVER_VIRTIO
	}
	gn.Driver = driver
	gn.NumQueues = numQueues
	gn.RxTrafficLimit = args.rxTrafficLimit
	gn.TxTrafficLimit = args.txTrafficLimit
	if bwLimit >= 0 {
		gn.BwLimit = bwLimit
	}

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)

	vpc, _ := network.GetVpc()
	if vpc == nil {
		return nil, fmt.Errorf("cannot find vpc of network %s(%s)", network.Id, network.Name)
	}

	provider := vpc.GetProviderName()
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
		} else if provider == api.CLOUD_PROVIDER_ONECLOUD || options.Options.EnablePreAllocateIpAddr {
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

		if vpc.Id != api.DEFAULT_VPC_ID && provider == api.CLOUD_PROVIDER_ONECLOUD {
			var err error
			GuestnetworkManager.lockAllocMappedAddr(ctx)
			defer GuestnetworkManager.unlockAllocMappedAddr(ctx)
			gn.MappedIpAddr, err = GuestnetworkManager.allocMappedIpAddr(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	var err error
	if ipBindMac := NetworkIpMacManager.GetMacFromIp(network.Id, gn.IpAddr); ipBindMac != "" {
		gn.MacAddr = ipBindMac
	} else {
		gn.MacAddr, err = manager.GenerateMac(mac)
		if err != nil {
			return nil, err
		}
	}

	if len(gn.MacAddr) == 0 {
		log.Errorf("Mac address generate fails")
		return nil, fmt.Errorf("mac address generate fails")
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

func (gn *SGuestnetwork) generateIfname(network *SNetwork, virtual bool, randomized bool) string {
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
		ip, _ := netutils.NewIPV4Addr(gn.IpAddr)
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

func (gn *SGuestnetwork) checkOrAllocateIfname(network *SNetwork, preferIfname string) (string, error) {
	man := GuestnetworkManager
	if !man.ifnameUsed(preferIfname) {
		return preferIfname, nil
	}

	ifname := gn.generateIfname(network, gn.Virtual, false)
	if !man.ifnameUsed(ifname) {
		return ifname, nil
	}
	if !gn.Virtual {
		ifname = gn.generateIfname(network, true, false)
	}
	found := false
	for i := 0; i < 5; i++ {
		if !man.ifnameUsed(ifname) {
			found = true
			break
		}
		ifname = gn.generateIfname(network, true, true)
	}
	if !found {
		return "", httperrors.NewConflictError("cannot allocate ifname")
	}
	return ifname, nil
}

func (gn *SGuestnetwork) GetGuest() *SGuest {
	guest, _ := GuestManager.FetchById(gn.GuestId)
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

func (gn *SGuestnetwork) GetTeamGuestnetwork() (*SGuestnetwork, error) {
	if len(gn.TeamWith) > 0 {
		return GuestnetworkManager.FetchByIdsAndIpMac(gn.GuestId, gn.NetworkId, "", gn.TeamWith)
	}
	return nil, nil
}

func (gn *SGuestnetwork) getJsonDescAtBaremetal(host *SHost) *api.GuestnetworkJsonDesc {
	net := gn.GetNetwork()
	hostwire := host.getHostwireOfIdAndMac(net.WireId, gn.MacAddr)
	return gn.getJsonDescHostwire(hostwire)
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

func (gn *SGuestnetwork) getJsonDescAtHost(ctx context.Context, host *SHost) *api.GuestnetworkJsonDesc {
	var (
		ret     *api.GuestnetworkJsonDesc = nil
		network                           = gn.GetNetwork()
	)
	if network.isOneCloudVpcNetwork() {
		ret = gn.getJsonDescOneCloudVpc(network)
	} else {
		hostWire, err := guestGetHostWireFromNetwork(host, network)
		if err != nil {
			log.Errorln(err)
		}
		ret = gn.getJsonDescHostwire(hostWire)
	}
	{
		ipnets, err := NetworkAddressManager.fetchAddressesByGuestnetworkId(gn.RowId)
		if err != nil {
			log.Errorln(err)
		}
		if len(ipnets) > 0 {
			ret.Networkaddresses = jsonutils.Marshal(ipnets)
		}
	}
	return ret
}

func (gn *SGuestnetwork) getJsonDescHostwire(hostwire *SHostwire) *api.GuestnetworkJsonDesc {
	desc := gn.getJsonDesc()
	if hostwire != nil {
		desc.Bridge = hostwire.Bridge
		desc.WireId = hostwire.WireId
		desc.Interface = hostwire.Interface
	}
	return desc
}

func (gn *SGuestnetwork) getJsonDescOneCloudVpc(network *SNetwork) *api.GuestnetworkJsonDesc {
	if gn.MappedIpAddr == "" {
		var (
			err  error
			addr string
		)
		addr, err = GuestnetworkManager.allocMappedIpAddr(context.TODO())
		if err != nil {
			log.Errorf("getJsonDescOneCloudVpc: row %d: alloc mapped ipaddr: %v", gn.RowId, err)
		} else {
			if _, err := db.Update(gn, func() error {
				gn.MappedIpAddr = addr
				return nil
			}); err != nil {
				log.Errorf("getJsonDescOneCloudVpc: row %d: db update mapped addr: %v", gn.RowId, err)
				gn.MappedIpAddr = ""
			}
		}
	}

	desc := gn.getJsonDesc()

	vpc, _ := network.GetVpc()
	desc.Vpc.Id = vpc.Id
	desc.Vpc.Provider = api.VPC_PROVIDER_OVN
	desc.Vpc.MappedIpAddr = gn.MappedIpAddr

	return desc
}

func (gn *SGuestnetwork) getJsonDesc() *api.GuestnetworkJsonDesc {
	net := gn.GetNetwork()
	desc := &api.GuestnetworkJsonDesc{
		GuestnetworkBaseDesc: api.GuestnetworkBaseDesc{
			Net:     net.Name,
			NetId:   gn.NetworkId,
			Mac:     gn.MacAddr,
			Virtual: gn.Virtual,
		},
	}

	if gn.Virtual {
		if len(gn.TeamWith) > 0 {
			teamGN, _ := gn.GetTeamGuestnetwork()
			if teamGN != nil {
				desc.Ip = teamGN.IpAddr
			}
		} else {
			desc.Ip = net.GetNetAddr().String()
		}
	} else {
		desc.Ip = gn.IpAddr
	}
	desc.Gateway = net.GuestGateway
	desc.Dns = net.GetDNS()
	desc.Domain = net.GetDomain()
	desc.Ntp = net.GetNTP()

	routes := net.GetRoutes()
	if routes != nil && len(routes) > 0 {
		desc.Routes = jsonutils.Marshal(routes)
	}
	desc.Ifname = gn.Ifname
	desc.Masklen = net.GuestIpMask
	desc.Driver = gn.Driver
	desc.NumQueues = gn.NumQueues
	desc.RxTrafficLimit = gn.RxTrafficLimit
	desc.TxTrafficLimit = gn.TxTrafficLimit
	desc.Vlan = net.VlanId
	desc.Bw = gn.getBandwidth()
	desc.Mtu = gn.getMtu(net)
	desc.Index = gn.Index
	desc.VirtualIps = gn.GetVirtualIPs()
	desc.ExternalId = net.ExternalId
	desc.TeamWith = gn.TeamWith

	guest := gn.getGuest()
	if guest.GetHypervisor() != api.HYPERVISOR_KVM || gn.IsSriovWithoutOffload() {
		manual := true
		desc.Manual = &manual
	}

	return desc
}

func (gn *SGuestnetwork) IsSriovWithoutOffload() bool {
	if gn.Driver != api.NETWORK_DRIVER_VFIO {
		return false
	}
	if dev, _ := gn.GetIsolatedDevice(); dev != nil && dev.OvsOffloadInterface != "" {
		return false
	}
	return true
}

func (gn *SGuestnetwork) UpdateNicTrafficUsed(rx, tx int64) error {
	_, err := db.Update(gn, func() error {
		gn.RxTrafficUsed = rx
		gn.TxTrafficUsed = tx
		return nil
	})
	return err
}

func (gn *SGuestnetwork) UpdateNicTrafficLimit(rx, tx *int64) error {
	_, err := db.Update(gn, func() error {
		if rx != nil {
			gn.RxTrafficLimit = *rx
		}
		if tx != nil {
			gn.TxTrafficLimit = *tx
		}
		return nil
	})
	return err
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

func (gn *SGuestnetwork) GetDetailedString() string {
	network := gn.GetNetwork()
	naCount, _ := NetworkAddressManager.fetchAddressCountByGuestnetworkId(gn.RowId)
	return fmt.Sprintf("eth%d:%s/%d/%s/%d/%s/%s/%d/%d", gn.Index, gn.IpAddr, network.GuestIpMask,
		gn.MacAddr, network.VlanId, network.Name, gn.Driver, gn.getBandwidth(), naCount)
}

func (gn *SGuestnetwork) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestnetworkUpdateInput,
) (api.GuestnetworkUpdateInput, error) {
	if input.Index != nil {
		index := *input.Index
		q := GuestnetworkManager.Query().SubQuery()
		count, err := q.Query().Filter(sqlchemy.Equals(q.Field("guest_id"), gn.GuestId)).
			Filter(sqlchemy.NotEquals(q.Field("network_id"), gn.NetworkId)).
			Filter(sqlchemy.Equals(q.Field("index"), index)).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("checkout nic index uniqueness fail %s", err)
		}
		if count > 0 {
			return input, httperrors.NewDuplicateResourceError("NIC Index %d has been occupied", index)
		}
	}
	var err error
	input.GuestJointBaseUpdateInput, err = gn.SGuestJointsBase.ValidateUpdateData(ctx, userCred, query, input.GuestJointBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SGuestJointsBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SGuestnetworkManager) DeleteGuestNics(ctx context.Context, userCred mcclient.TokenCredential, gns []SGuestnetwork, reserve bool) error {
	for i := range gns {
		gn := gns[i]
		if len(gn.EipId) > 0 {
			return errors.Wrapf(httperrors.ErrInvalidStatus, "eip associate with %s", gn.IpAddr)
		}
		guest := gn.GetGuest()
		dev, err := guest.GetIsolatedDeviceByNetworkIndex(gn.Index)
		if err != nil {
			return errors.Wrap(err, "GetIsolatedDeviceByNetworkIndex")
		}
		net := gn.GetNetwork()
		if regutils.MatchIP4Addr(gn.IpAddr) || regutils.MatchIP6Addr(gn.Ip6Addr) {
			net.updateDnsRecord(&gn, false)
			if regutils.MatchIP4Addr(gn.IpAddr) {
				// ??
				// netman.get_manager().netmap_remove_node(gn.ip_addr)
			}
		}
		err = gn.Delete(ctx, userCred)
		if err != nil {
			log.Errorf("%s", err)
		}
		gn.LogDetachEvent(ctx, userCred, guest, net)
		if dev != nil {
			err = guest.detachIsolateDevice(ctx, userCred, dev)
			if err != nil {
				return err
			}
		}

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

func (gn *SGuestnetwork) LogDetachEvent(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, network *SNetwork) {
	if network == nil {
		netTmp, _ := NetworkManager.FetchById(gn.NetworkId)
		network = netTmp.(*SNetwork)
	}
	db.OpsLog.LogDetachEvent(ctx, guest, network, userCred, nil)
}

func (gn *SGuestnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := NetworkAddressManager.deleteByGuestnetworkId(ctx, userCred, gn.RowId); err != nil {
		return errors.Wrap(err, "delete attached network addresses")
	}
	return db.DeleteModel(ctx, userCred, gn)
}

func (gn *SGuestnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, gn)
}

func totalGuestNicCount(
	scope rbacscope.TRbacScope,
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
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(guests.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacscope.ScopeProject:
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

func (gn *SGuestnetwork) IsExit() bool {
	if gn.IpAddr != "" {
		addr, err := netutils.NewIPV4Addr(gn.IpAddr)
		if err == nil {
			return netutils.IsExitAddress(addr)
		}
	}
	net := gn.GetNetwork()
	if net != nil {
		return net.IsExitNetwork()
	}
	return false
}

func (gn *SGuestnetwork) getBandwidth() int {
	if gn.BwLimit == 0 {
		return 0
	}
	if gn.BwLimit > 0 && gn.BwLimit <= api.MAX_BANDWIDTH {
		return gn.BwLimit
	} else {
		net := gn.GetNetwork()
		if net != nil {
			wire, _ := net.GetWire()
			if wire != nil {
				return wire.Bandwidth
			}
		}
		return options.Options.DefaultBandwidth
	}
}

func (gn *SGuestnetwork) getMtu(net *SNetwork) int {
	return net.getMtu()
}

func (gn *SGuestnetwork) IsAllocated() bool {
	region, _ := gn.GetGuest().getRegion()
	provider := region.Provider
	if regutils.MatchMacAddr(gn.MacAddr) && (gn.Virtual || regutils.MatchIP4Addr(gn.IpAddr) || (provider != api.CLOUD_PROVIDER_ONECLOUD && !options.Options.EnablePreAllocateIpAddr)) {
		return true
	}
	return false
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

func (gn *SGuestnetwork) GetVirtualIPs() []string {
	ips := make([]string, 0)
	guest := gn.GetGuest()
	net := gn.GetNetwork()
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

func (gn *SGuestnetwork) GetIsolatedDevice() (*SIsolatedDevice, error) {
	dev := SIsolatedDevice{}
	q := IsolatedDeviceManager.Query().Equals("guest_id", gn.GuestId).Equals("network_index", gn.Index)
	if cnt, err := q.CountWithError(); err != nil {
		return nil, err
	} else if cnt == 0 {
		return nil, nil
	}
	err := q.First(&dev)
	if err != nil {
		return nil, err
	}
	dev.SetModelManager(IsolatedDeviceManager, &dev)
	return &dev, nil
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

func (manager *SGuestnetworkManager) FetchByGuestId(guestId string) ([]SGuestnetwork, error) {
	q := manager.Query().
		Equals("guest_id", guestId)
	q = q.Asc(q.Field("index"))
	var rets []SGuestnetwork
	if err := db.FetchModelObjects(manager, q, &rets); err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return rets, nil
}

func (manager *SGuestnetworkManager) FetchByGuestIdIndex(guestId string, index int8) (*SGuestnetwork, error) {
	rets, err := manager.FetchByGuestId(guestId)
	if err != nil {
		return nil, errors.Wrap(err, "FetchByGuestId")
	}
	if index >= 0 && int(index) < len(rets) {
		return &rets[index], nil
	}
	return nil, errors.ErrNotFound
}

func (gn *SGuestnetwork) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := api.GuestnetworkShortDesc{}
	if len(gn.IpAddr) > 0 {
		desc.IpAddr = gn.IpAddr
		desc.IsExit = gn.IsExit()
	}
	if len(gn.Ip6Addr) > 0 {
		desc.Ip6Addr = gn.Ip6Addr
	}
	desc.Mac = gn.MacAddr
	if len(gn.TeamWith) > 0 {
		desc.TeamWith = gn.TeamWith
	}
	return jsonutils.Marshal(desc).(*jsonutils.JSONDict)
}

func (gn *SGuestnetwork) ToNetworkConfig() *api.NetworkConfig {
	net := gn.GetNetwork()
	if net == nil {
		return nil
	}
	wire, _ := net.GetWire()
	ret := &api.NetworkConfig{
		Index:   int(gn.Index),
		Network: net.Id,
		Wire:    wire.Id,
		Mac:     gn.MacAddr,
		Address: gn.IpAddr,
		Driver:  gn.Driver,
		BwLimit: gn.BwLimit,
		Project: net.ProjectId,
		Domain:  net.DomainId,
		Ifname:  gn.Ifname,
		NetType: net.ServerType,
		Exit:    net.IsExitNetwork(),
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
		naSubQ := NetworkAddressManager.Query("parent_id").Equals("type", api.NetworkAddressTypeSubIP).Equals("parent_type", api.NetworkAddressParentTypeGuestnetwork).In("ip_addr", query.IpAddr).SubQuery()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("ip_addr"), query.IpAddr),
			sqlchemy.In(q.Field("row_id"), naSubQ),
		))
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
