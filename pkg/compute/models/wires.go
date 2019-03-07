package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SWireManager struct {
	db.SStandaloneResourceBaseManager
}

var WireManager *SWireManager

func init() {
	WireManager = &SWireManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SWire{},
			"wires_tbl",
			"wire",
			"wires",
		),
	}
	WireManager.NameLength = 9
	WireManager.NameRequireAscii = true
}

type SWire struct {
	db.SStandaloneResourceBase

	Bandwidth    int    `list:"admin" update:"admin" nullable:"false" create:"admin_required"`            // = Column(Integer, nullable=False) # bandwidth of network in Mbps
	ScheduleRank int    `list:"admin" update:"admin"`                                                     // = Column(Integer, default=0, nullable=True)
	ZoneId       string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"admin_required"` // = Column(VARCHAR(36, charset='ascii'), nullable=False)
	VpcId        string `wdith:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
}

func (manager *SWireManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{ZoneManager, VpcManager}
}

func (self *SWireManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SWireManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SWire) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SWire) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SWire) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SWireManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	bandwidth, err := data.Int("bandwidth")
	if err != nil || bandwidth == 0 {
		return nil, httperrors.NewInputParameterError("invalid bandwidth")
	}

	vpcStr := jsonutils.GetAnyString(data, []string{"vpc", "vpc_id"})
	if len(vpcStr) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc_id")
	}

	if len(vpcStr) > 0 {
		vpcObj, err := VpcManager.FetchByIdOrName(userCred, vpcStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("Vpc %s not found", vpcStr)
			} else {
				return nil, httperrors.NewInternalServerError("Fetch Vpc %s error %s", vpcStr, err)
			}
		}
		data.Add(jsonutils.NewString(vpcObj.GetId()), "vpc_id")
	}

	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (wire *SWire) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	bandwidth, err := data.Int("bandwidth")
	if err == nil && bandwidth <= 0 {
		return nil, httperrors.NewInputParameterError("invalid bandwidth")
	}
	return wire.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (wire *SWire) ValidateDeleteCondition(ctx context.Context) error {
	if wire.HostCount() > 0 || wire.NetworkCount() > 0 {
		return httperrors.NewNotEmptyError("not an empty wire")
	}
	return wire.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (wire *SWire) getHostwireQuery() *sqlchemy.SQuery {
	return HostwireManager.Query().Equals("wire_id", wire.Id)
}

func (wire *SWire) HostCount() int {
	q := wire.getHostwireQuery()
	return q.Count()
}

func (wire *SWire) GetHostwires() ([]SHostwire, error) {
	q := wire.getHostwireQuery()
	hostwires := make([]SHostwire, 0)
	err := db.FetchModelObjects(HostwireManager, q, &hostwires)
	if err != nil {
		return nil, err
	}
	return hostwires, nil
}

func (wire *SWire) NetworkCount() int {
	q := NetworkManager.Query().Equals("wire_id", wire.Id)
	return q.Count()
}

func (wire *SWire) GetVpcId() string {
	if len(wire.VpcId) == 0 {
		return "default"
	} else {
		return wire.VpcId
	}
}

func (manager *SWireManager) getWiresByVpcAndZone(vpc *SVpc, zone *SZone) ([]SWire, error) {
	wires := make([]SWire, 0)
	q := manager.Query()
	if vpc != nil {
		q = q.Equals("vpc_id", vpc.Id)
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	err := db.FetchModelObjects(manager, q, &wires)
	if err != nil {
		log.Errorf("getWiresByVpcAndZone error %s", err)
		return nil, err
	}
	return wires, nil
}

func (manager *SWireManager) SyncWires(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc, wires []cloudprovider.ICloudWire) ([]SWire, []cloudprovider.ICloudWire, compare.SyncResult) {
	localWires := make([]SWire, 0)
	remoteWires := make([]cloudprovider.ICloudWire, 0)
	syncResult := compare.SyncResult{}

	dbWires, err := manager.getWiresByVpcAndZone(vpc, nil)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SWire, 0)
	commondb := make([]SWire, 0)
	commonext := make([]cloudprovider.ICloudWire, 0)
	added := make([]cloudprovider.ICloudWire, 0)

	err = compare.CompareSets(dbWires, wires, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].markNetworkUnknown(userCred)
		if err != nil { // cannot delete
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
		/* err = removed[i].ValidateDeleteCondition(ctx)
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
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudWire(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localWires = append(localWires, commondb[i])
			remoteWires = append(remoteWires, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudWire(added[i], vpc)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localWires = append(localWires, *new)
			remoteWires = append(remoteWires, added[i])
			syncResult.Add()
		}
	}

	return localWires, remoteWires, syncResult
}

func (self *SWire) syncWithCloudWire(extWire cloudprovider.ICloudWire) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = extWire.GetName()
		self.Bandwidth = extWire.GetBandwidth() // 10G

		self.IsEmulated = extWire.IsEmulated()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudWire error %s", err)
	}
	return err
}

func (self *SWire) markNetworkUnknown(userCred mcclient.TokenCredential) error {
	nets, err := self.getNetworks()
	if err != nil {
		return err
	}
	for i := 0; i < len(nets); i += 1 {
		nets[i].SetStatus(userCred, NETWORK_STATUS_UNKNOWN, "wire sync to remove")
	}
	return nil
}

func (manager *SWireManager) newFromCloudWire(extWire cloudprovider.ICloudWire, vpc *SVpc) (*SWire, error) {
	wire := SWire{}
	wire.SetModelManager(manager)

	wire.Name = extWire.GetName()
	wire.ExternalId = extWire.GetGlobalId()
	wire.Bandwidth = extWire.GetBandwidth()
	wire.VpcId = vpc.Id
	izone := extWire.GetIZone()
	if izone != nil {
		zoneObj, err := ZoneManager.FetchByExternalId(izone.GetGlobalId())
		if err != nil {
			log.Errorf("cannot find zone for wire %s", err)
			return nil, err
		}

		wire.ZoneId = zoneObj.(*SZone).Id
	}

	wire.IsEmulated = extWire.IsEmulated()

	err := manager.TableSpec().Insert(&wire)
	if err != nil {
		log.Errorf("newFromCloudWire fail %s", err)
		return nil, err
	}
	return &wire, nil
}

func (manager *SWireManager) totalCountQ(rangeObj db.IStandaloneModel, hostTypes []string, providers []string) *sqlchemy.SQuery {
	guests := GuestManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	gNics := GuestnetworkManager.Query().SubQuery()
	gNicQ := gNics.Query(
		gNics.Field("network_id"),
		sqlchemy.COUNT("id").Label("gnic_count"),
	)
	gNicQ = gNicQ.Join(guests, sqlchemy.Equals(guests.Field("id"), gNics.Field("guest_id")))
	gNicQ = gNicQ.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
	gNicQ = gNicQ.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

	hNics := HostnetworkManager.Query().SubQuery()
	hNicQ := hNics.Query(
		hNics.Field("network_id"),
		sqlchemy.COUNT("id").Label("hnic_count"),
	)
	hNicQ = hNicQ.Join(hosts, sqlchemy.Equals(hNics.Field("baremetal_id"), hosts.Field("id")))
	hNicQ = hNicQ.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

	revIps := ReservedipManager.Query().SubQuery()
	revQ := revIps.Query(revIps.Field("network_id"), sqlchemy.COUNT("id").Label("rnic_count"))

	gNicSQ := gNicQ.GroupBy(gNics.Field("network_id")).SubQuery()
	hNicSQ := hNicQ.GroupBy(hNics.Field("network_id")).SubQuery()
	revSQ := revQ.GroupBy(revIps.Field("network_id")).SubQuery()

	networks := NetworkManager.Query().SubQuery()
	netQ := networks.Query(
		networks.Field("wire_id"),
		sqlchemy.COUNT("id").Label("net_count"),
		sqlchemy.SUM("gnic_count", gNicQ.Field("gnic_count")),
		sqlchemy.SUM("hnic_count", hNicQ.Field("hnic_count")),
		sqlchemy.SUM("rev_count", revQ.Field("rnic_count")))
	netQ = netQ.LeftJoin(gNicSQ, sqlchemy.Equals(gNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(hNicSQ, sqlchemy.Equals(hNicSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.LeftJoin(revSQ, sqlchemy.Equals(revSQ.Field("network_id"), networks.Field("id")))
	netQ = netQ.GroupBy(networks.Field("wire_id"))
	netSQ := netQ.SubQuery()

	wires := WireManager.Query().SubQuery()
	q := wires.Query(
		sqlchemy.COUNT("id").Label("wires_count"),
		sqlchemy.SUM("net_count", netSQ.Field("net_count")),
		sqlchemy.SUM("guest_nic_count", netSQ.Field("gnic_count")),
		sqlchemy.SUM("host_nic_count", netSQ.Field("hnic_count")),
		sqlchemy.SUM("reserved_count", netSQ.Field("rev_count")),
	)
	q = q.LeftJoin(netSQ, sqlchemy.Equals(wires.Field("id"), netSQ.Field("wire_id")))

	if rangeObj != nil || len(hostTypes) > 0 {
		hostwires := HostwireManager.Query().SubQuery()
		sq := hostwires.Query(hostwires.Field("wire_id"))
		sq = sq.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
		sq = sq.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
		sq = AttachUsageQuery(sq, hosts, hostTypes, nil, nil, rangeObj)
		q = q.Filter(sqlchemy.In(wires.Field("id"), sq.Distinct()))
	}

	if len(providers) > 0 {
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := vpcs.Query(vpcs.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(vpcs.Field("manager_id"), cloudproviders.Field("id")))
		subq = subq.Filter(sqlchemy.In(cloudproviders.Field("provider"), providers))
		q = q.Filter(sqlchemy.In(wires.Field("vpc_id"), subq.SubQuery()))
	}

	return q
}

type WiresCountStat struct {
	WiresCount    int
	NetCount      int
	GuestNicCount int
	HostNicCount  int
	ReservedCount int
}

func (manager *SWireManager) TotalCount(rangeObj db.IStandaloneModel, hostTypes []string, providers []string) WiresCountStat {
	stat := WiresCountStat{}
	err := manager.totalCountQ(rangeObj, hostTypes, providers).First(&stat)
	if err != nil {
		log.Errorf("Wire total count: %v", err)
	}
	return stat
}

func (self *SWire) getNetworkQuery() *sqlchemy.SQuery {
	return NetworkManager.Query().Equals("wire_id", self.Id)
}

func (self *SWire) getNetworks() ([]SNetwork, error) {
	q := self.getNetworkQuery()
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (self *SWire) getGatewayNetworkQuery() *sqlchemy.SQuery {
	q := self.getNetworkQuery()
	q = q.IsNotNull("guest_gateway").IsNotEmpty("guest_gateway")
	q = q.Equals("status", NETWORK_STATUS_AVAILABLE)
	return q
}

func (self *SWire) getPublicNetworks() ([]SNetwork, error) {
	q := self.getGatewayNetworkQuery()
	q = q.IsTrue("is_public")
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (self *SWire) getPrivateNetworks(userCred mcclient.TokenCredential) ([]SNetwork, error) {
	q := self.getGatewayNetworkQuery()
	q = q.Equals("tenant_id", userCred.GetProjectId()).IsFalse("is_public")
	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	return nets, nil
}

func (self *SWire) GetCandidatePrivateNetwork(userCred mcclient.TokenCredential, isExit bool, serverTypes []string) (*SNetwork, error) {
	nets, err := self.getPrivateNetworks(userCred)
	if err != nil {
		return nil, err
	}
	return ChooseCandidateNetworks(nets, isExit, serverTypes), nil
}

func (self *SWire) GetCandidatePublicNetwork(isExit bool, serverTypes []string) (*SNetwork, error) {
	nets, err := self.getPublicNetworks()
	if err != nil {
		return nil, err
	}
	return ChooseCandidateNetworks(nets, isExit, serverTypes), nil
}

func (self *SWire) GetCandidateNetworkForIp(userCred mcclient.TokenCredential, ipAddr string) (*SNetwork, error) {
	ip, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return nil, err
	}
	netPrivates, err := self.getPrivateNetworks(userCred)
	if err != nil {
		return nil, err
	}
	for _, net := range netPrivates {
		if net.isAddressInRange(ip) {
			return &net, nil
		}
	}
	netPublics, err := self.getPublicNetworks()
	if err != nil {
		return nil, err
	}
	for _, net := range netPublics {
		if net.isAddressInRange(ip) {
			return &net, nil
		}
	}
	return nil, nil
}

func chooseNetworkByAddressCount(nets []*SNetwork) (*SNetwork, *SNetwork) {
	minCnt := 65535
	maxCnt := 0
	var minSel *SNetwork
	var maxSel *SNetwork
	for _, net := range nets {
		cnt := net.getFreeAddressCount()
		if cnt <= 0 {
			continue
		}
		if minSel == nil || minCnt > cnt {
			minSel = net
			minCnt = cnt
		}
		if maxSel == nil || maxCnt < cnt {
			maxSel = net
			maxCnt = cnt
		}
	}
	return minSel, maxSel
}

func ChooseCandidateNetworks(nets []SNetwork, isExit bool, serverTypes []string) *SNetwork {
	for _, s := range serverTypes {
		net := chooseCandidateNetworksByNetworkType(nets, isExit, s)
		if net != nil {
			return net
		}
	}
	return nil
}

func chooseCandidateNetworksByNetworkType(nets []SNetwork, isExit bool, serverType string) *SNetwork {
	matchingNets := make([]*SNetwork, 0)
	notMatchingNets := make([]*SNetwork, 0)

	for i := 0; i < len(nets); i++ {
		net := nets[i]
		if isExit != net.IsExitNetwork() {
			continue
		}
		if serverType == net.ServerType || (len(net.ServerType) == 0 && serverType == NETWORK_TYPE_GUEST) {
			matchingNets = append(matchingNets, &net)
		} else {
			notMatchingNets = append(notMatchingNets, &net)
		}
	}
	minSel, maxSel := chooseNetworkByAddressCount(matchingNets)
	if (isExit && minSel == nil) || (!isExit && maxSel == nil) {
		minSel, maxSel = chooseNetworkByAddressCount(notMatchingNets)
	}
	if isExit {
		return minSel
	} else {
		return maxSel
	}
}

func (self *SWire) GetZone() *SZone {
	return ZoneManager.FetchZoneById(self.ZoneId)
}

func (manager *SWireManager) InitializeData() error {
	wires := make([]SWire, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &wires)
	if err != nil {
		return err
	}
	for _, w := range wires {
		if len(w.VpcId) == 0 {
			manager.TableSpec().Update(&w, func() error {
				w.VpcId = "default"
				return nil
			})
		}
	}
	return nil
}

func (wire *SWire) getEnabledHosts() []SHost {
	hosts := make([]SHost, 0)

	hostQuery := HostManager.Query().SubQuery()
	hostwireQuery := HostwireManager.Query().SubQuery()

	q := hostQuery.Query()
	q = q.Join(hostwireQuery, sqlchemy.AND(sqlchemy.Equals(hostQuery.Field("id"), hostwireQuery.Field("host_id")),
		sqlchemy.IsFalse(hostwireQuery.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hostwireQuery.Field("wire_id"), wire.Id))
	q = q.Filter(sqlchemy.IsTrue(hostQuery.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(hostQuery.Field("host_status"), HOST_ONLINE))

	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		log.Errorf("getEnabledHosts fail %s", err)
		return nil
	}

	return hosts
}

func (wire *SWire) clearHostSchedDescCache() error {
	hosts := wire.getEnabledHosts()
	if hosts != nil {
		for i := 0; i < len(hosts); i += 1 {
			err := hosts[i].ClearSchedDescCache()
			if err != nil {
				log.Errorf("%s", err)
				return err
			}
		}
	}
	return nil
}

func (wire *SWire) getVpc() *SVpc {
	vpcObj, err := VpcManager.FetchById(wire.VpcId)
	if err != nil {
		log.Errorf("getVpc fail %s", err)
		return nil
	}
	return vpcObj.(*SVpc)
}

func (self *SWire) GetIWire() (cloudprovider.ICloudWire, error) {
	vpc := self.getVpc()
	if vpc == nil {
		log.Errorf("Cannot find VPC for wire???")
		return nil, fmt.Errorf("No VPC?????")
	}
	ivpc, err := vpc.GetIVpc()
	if err != nil {
		return nil, err
	}
	return ivpc.GetIWireById(self.GetExternalId())
}

func (manager *SWireManager) FetchWireById(wireId string) *SWire {
	wireObj, err := manager.FetchById(wireId)
	if err != nil {
		log.Errorf("FetchWireById fail %s", err)
		return nil
	}
	return wireObj.(*SWire)
}

func (manager *SWireManager) GetOnPremiseWireOfIp(ipAddr string) (*SWire, error) {
	net, err := NetworkManager.GetOnPremiseNetworkOfIP(ipAddr, "", tristate.None)
	if err != nil {
		return nil, err
	}
	wire := net.GetWire()
	if wire != nil {
		return wire, nil
	} else {
		return nil, fmt.Errorf("Wire not found")
	}
}

func (manager *SWireManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error

	q, err = managedResourceFilterByAccount(q, query, "vpc_id", func() *sqlchemy.SQuery {
		vpcs := VpcManager.Query().SubQuery()
		subq := vpcs.Query(vpcs.Field("id"))
		return subq
	})
	if err != nil {
		return nil, err
	}

	q = managedResourceFilterByCloudType(q, query, "vpc_id", func() *sqlchemy.SQuery {
		vpcs := VpcManager.Query().SubQuery()
		subq := vpcs.Query(vpcs.Field("id"))
		return subq
	})

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	vpcStr := jsonutils.GetAnyString(query, []string{"vpc_id", "vpc"})
	if len(vpcStr) > 0 {
		vpc, err := VpcManager.FetchByIdOrName(userCred, vpcStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("vpc %s not found", vpcStr)
			} else {
				return nil, httperrors.NewInternalServerError("vpc %s query fail %s", vpcStr, err)
			}
		}
		q = q.Equals("vpc_id", vpc.GetId())
	}

	regionStr := jsonutils.GetAnyString(query, []string{"region_id", "region", "cloudregion_id", "cloudregion"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("region %s not found", regionStr)
			} else {
				return nil, httperrors.NewInternalServerError("region %s query fail %s", regionStr, err)
			}
		}
		sq := VpcManager.Query("id").Equals("cloudregion_id", region.GetId())
		q = q.In("vpc_id", sq.SubQuery())
	}

	/*managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		sq := VpcManager.Query("id").Equals("manager_id", provider.GetId())
		q = q.In("vpc_id", sq.SubQuery())
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := vpcs.Query(vpcs.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), vpcs.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), account.GetId()))
		q = q.Filter(sqlchemy.In(q.Field("vpc_id"), subq.SubQuery()))
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		vpcs := VpcManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := vpcs.Query(vpcs.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), vpcs.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("provider"), providerStr))

		q = q.Filter(sqlchemy.In(q.Field("vpc_id"), subq.SubQuery()))
	}*/

	return q, err
}

func (self *SWire) getRegion() *SCloudregion {
	zone := self.GetZone()
	if zone != nil {
		return zone.GetRegion()
	}
	return nil
}

func (self *SWire) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SWire) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SWire) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.NetworkCount())), "networks")
	/*zone := self.GetZone()
	if zone != nil {
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
	}*/
	vpc := self.getVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.GetName()), "vpc")
		if len(vpc.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(vpc.GetExternalId()), "vpc_ext_id")
		}
	}

	info := vpc.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	return extra
}
