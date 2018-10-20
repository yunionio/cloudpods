package models

import (
	"context"
	"database/sql"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"
)

const (
	ZONE_ENABLE  = "enable"
	ZONE_DISABLE = "disable"
	ZONE_SOLDOUT = "soldout"
	ZONE_LACK    = "lack"
)

type SZoneManager struct {
	db.SStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var ZoneManager *SZoneManager

func init() {
	ZoneManager = &SZoneManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SZone{},
			"zones_tbl",
			"zone",
			"zones",
		),
	}
	ZoneManager.NameRequireAscii = false
}

type SZone struct {
	db.SStatusStandaloneResourceBase
	SInfrastructure

	Location string `width:"256" charset:"utf8" get:"user" update:"admin"` // = Column(VARCHAR(256, charset='utf8'))
	Contacts string `width:"256" charset:"utf8" get:"user" update:"admin"` // = Column(VARCHAR(256, charset='utf8'))
	NameCn   string `width:"256" charset:"utf8"`                           // = Column(VARCHAR(256, charset='utf8'))
	// status = Column(VARCHAR(36, charset='ascii'), nullable=False, default=ZONE_DISABLE)
	ManagerUri string `width:"256" charset:"ascii" list:"admin" update:"admin"` // = Column(VARCHAR(256, charset='ascii'), nullable=True)
	// admin_id = Column(VARCHAR(36, charset='ascii'), nullable=False)
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`
}

func (manager *SZoneManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{CloudregionManager}
}

func (manager *SZoneManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (zone *SZone) ValidateDeleteCondition(ctx context.Context) error {
	usage := zone.GeneralUsage()
	if !usage.isEmpty() {
		return httperrors.NewNotEmptyError("not empty zone")
	}
	return zone.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

/*
@classmethod
def tenant_id_hash(cls, tenant_id, mod):
intval = 0
for i in range(len(tenant_id)):
intval += ord(tenant_id[i])
return intval % mod

@classmethod
def get_hashed_zone_id(cls, tenant_id, excludes=None):
from clouds.models.hosts    import Hosts
q = Hosts.query(Hosts.zone_id, func.count('*')) \
.filter(Hosts.enabled==True) \
.filter(Hosts.host_status==Hosts.HOST_ONLINE)
if excludes is not None and len(excludes) > 0:
q = q.filter(not_(Hosts.zone_id.in_(excludes)))
q = q.group_by(Hosts.zone_id).all()
zones = []
weights = {}
for (zone_id, weight) in q:
zones.append(zone_id)
weights[zone_id] = weight
ring = HashRing(zones, weights)
return ring.get_node(tenant_id)


def is_zone_manageable(self):
if self.manager_uri is not None and len(self.manager_uri) > 0:
return True
else:
return False

def request(self, url, on_succ, on_fail, user_cred, **kwargs):
headers = {}
headers['X-Auth-Token'] = user_cred.token
zoneclient.get_client().request(self, 'GET', url, headers, \
on_succ, on_fail, **kwargs)

*/

func (manager *SZoneManager) Count() int {
	return manager.Query().Count()
}

type ZoneGeneralUsage struct {
	Hosts             int
	HostsEnabled      int
	Baremetals        int
	BaremetalsEnabled int
	Wires             int
	Networks          int
	Storages          int
}

func (usage *ZoneGeneralUsage) isEmpty() bool {
	if usage.Hosts > 0 {
		return false
	}
	if usage.Wires > 0 {
		return false
	}
	if usage.Networks > 0 {
		return false
	}
	if usage.Storages > 0 {
		return false
	}
	return true
}

func (zone *SZone) GeneralUsage() ZoneGeneralUsage {
	usage := ZoneGeneralUsage{}
	usage.Hosts = zone.HostCount("", "", tristate.None, "", tristate.None)
	usage.HostsEnabled = zone.HostCount("", "", tristate.True, "", tristate.None)
	usage.Baremetals = zone.HostCount("", "", tristate.None, "", tristate.True)
	usage.BaremetalsEnabled = zone.HostCount("", "", tristate.True, "", tristate.True)
	usage.Wires = zone.getWireCount()
	usage.Networks = zone.getNetworkCount()
	usage.Storages = zone.getStorageCount()
	return usage
}

func (zone *SZone) HostCount(status string, hostStatus string, enabled tristate.TriState, hostType string, isBaremetal tristate.TriState) int {
	q := HostManager.Query().Equals("zone_id", zone.Id)
	if len(status) > 0 {
		q = q.Equals("status", status)
	}
	if len(hostStatus) > 0 {
		q = q.Equals("host_status", hostStatus)
	}
	if enabled.IsTrue() {
		q = q.IsTrue("enabled")
	} else if enabled.IsFalse() {
		q = q.IsFalse("enabled")
	}
	if len(hostType) > 0 {
		q = q.Equals("host_type", hostType)
	}
	if isBaremetal.IsTrue() {
		q = q.IsTrue("is_baremetal")
	} else if isBaremetal.IsFalse() {
		q = q.IsFalse("is_baremetal")
	}
	return q.Count()
}

func (zone *SZone) getWireCount() int {
	q := WireManager.Query().Equals("zone_id", zone.Id)
	return q.Count()
}

func (zone *SZone) getStorageCount() int {
	q := StorageManager.Query().Equals("zone_id", zone.Id)
	return q.Count()
}

func (zone *SZone) getNetworkCount() int {
	return getNetworkCount(zone)
}

/*def host_count(self, status=None, host_status=None, enabled=None, host_type=None, is_baremetal=None):
from hosts import Hosts
q = Hosts.query().filter(Hosts.zone_id==self.id)
if status is not None:
q = q.filter(Hosts.status==status)
if host_status is not None:
q = q.filter(Hosts.host_status==host_status)
if enabled is not None:
q = q.filter(Hosts.enabled==enabled)
if host_type is not None:
q = q.filter(Hosts.host_type==host_type)
if is_baremetal is not None:
q = q.filter(Hosts.is_baremetal==is_baremetal)
return q.count()

def wire_count(self):
from wires import Wires
q = Wires.query().filter(Wires.zone_id==self.id)
return q.count()
*/

/*

@classmethod
def get_resident_zones_of_tenant(cls, tenant_id):
from guests import Guests
from hosts import Hosts
q = Zones.query(Zones.id, func.count('*')) \
.join(Hosts, and_(Hosts.zone_id==Zones.id,
Hosts.deleted==False,
Hosts.enabled==True,
Hosts.host_status==Hosts.HOST_ONLINE)) \
.join(Guests, and_(Guests.host_id==Hosts.id,
Guests.deleted==False)) \
.filter(Guests.tenant_id==tenant_id) \
.group_by(Zones.id)
ret = []
for (zone_id, guest_cnt) in q.all():
ret.append((Zones.fetch(zone_id), guest_cnt))
return ret

@classmethod
def get_default_schedule_zone_of_tenant(cls, user_cred, tenant_id, params,
count=1):
from guests import Guests
zones = cls.get_resident_zones_of_tenant(tenant_id)
if len(zones) == 0:
if options.default_zone is not None:
zone = Zones.fetch(options.default_zone)
ret = Guests.check_resource_limit(user_cred, params, count,
range_obj=zone)
if len(ret) > 0:
logging.error('Default zone out of resources')
return None
else:
return zone
else:
sel_z = None
max_cnt = 0
for (z, gcnt) in zones:
ret = Guests.check_resource_limit(user_cred, params, count,
range_obj=z)
if len(ret) == 0:
if max_cnt < gcnt:
sel_z = z
max_cnt = gcnt
return sel_z

def get_wires(self):
from wires import Wires
q = Wires.query().filter(Wires.zone_id==self.id)
return q.all()

def get_name_cn(self):
return self.name_cn or self.name


def get_more_details(self):
ret = {}
ret.update(self.general_usage(None))
return ret

def get_customize_columns(self, user_cred, kwargs):
ret = super(Zones, self).get_customize_columns(user_cred, kwargs)
ret.update(self.get_more_details())
return ret

def get_details(self, user_cred, kwargs):
ret = super(Zones, self).get_details(user_cred, kwargs)
ret.update(self.get_more_details())
return ret

*/

func zoneExtra(zone *SZone, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	usage := zone.GeneralUsage()
	extra.Update(jsonutils.Marshal(usage))
	region := zone.GetRegion()
	if region != nil {
		extra.Add(jsonutils.NewString(region.Name), "cloudregion")
	}
	return extra
}

func (zone *SZone) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := zone.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return zoneExtra(zone, extra)
}

func (zone *SZone) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := zone.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return zoneExtra(zone, extra)
}

func (zone *SZone) GetCloudRegionId() string {
	if len(zone.CloudregionId) == 0 {
		return "default"
	} else {
		return zone.CloudregionId
	}
}

func (manager *SZoneManager) getZonesByRegion(region *SCloudregion) ([]SZone, error) {
	zones := make([]SZone, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	err := db.FetchModelObjects(manager, q, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (manager *SZoneManager) SyncZones(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zones []cloudprovider.ICloudZone) ([]SZone, []cloudprovider.ICloudZone, compare.SyncResult) {
	localZones := make([]SZone, 0)
	remoteZones := make([]cloudprovider.ICloudZone, 0)
	syncResult := compare.SyncResult{}

	dbZones, err := manager.getZonesByRegion(region)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SZone, 0)
	commondb := make([]SZone, 0)
	commonext := make([]cloudprovider.ICloudZone, 0)
	added := make([]cloudprovider.ICloudZone, 0)

	err = compare.CompareSets(dbZones, zones, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, ZONE_DISABLE, "sync to delete")
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudZone(commonext[i], region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localZones = append(localZones, commondb[i])
			remoteZones = append(remoteZones, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudZone(added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localZones = append(localZones, *new)
			remoteZones = append(remoteZones, added[i])
			syncResult.Add()
		}
	}

	return localZones, remoteZones, syncResult
}

func (self *SZone) syncWithCloudZone(extZone cloudprovider.ICloudZone, region *SCloudregion) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = extZone.GetName()
		self.Status = extZone.GetStatus()

		self.IsEmulated = extZone.IsEmulated()

		self.CloudregionId = region.Id

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}
	return err
}

func (manager *SZoneManager) newFromCloudZone(extZone cloudprovider.ICloudZone, region *SCloudregion) (*SZone, error) {
	zone := SZone{}
	zone.SetModelManager(manager)

	zone.Name = extZone.GetName()
	zone.Status = extZone.GetStatus()
	zone.ExternalId = extZone.GetGlobalId()

	zone.IsEmulated = extZone.IsEmulated()

	zone.CloudregionId = region.Id

	err := manager.TableSpec().Insert(&zone)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}
	return &zone, nil
}

func (manager *SZoneManager) FetchZoneById(zoneId string) *SZone {
	zoneObj, err := manager.FetchById(zoneId)
	if err != nil {
		log.Errorf("%s", err)
	}
	return zoneObj.(*SZone)
}

func (zone *SZone) GetRegion() *SCloudregion {
	return CloudregionManager.FetchRegionById(zone.GetCloudRegionId())
}

func (manager *SZoneManager) InitializeData() error {
	// set cloudregion ID
	zones := make([]SZone, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &zones)
	if err != nil {
		return err
	}
	for _, z := range zones {
		if len(z.CloudregionId) == 0 {
			manager.TableSpec().Update(&z, func() error {
				z.CloudregionId = "default"
				return nil
			})
		}
	}
	return nil
}

func (manager *SZoneManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	if jsonutils.QueryBoolean(query, "is_private", false) {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("external_id")),
			sqlchemy.IsEmpty(q.Field("external_id"))))
	}
	if jsonutils.QueryBoolean(query, "is_public", false) {
		q = q.Filter(sqlchemy.AND(sqlchemy.IsNotNull(q.Field("external_id")),
			sqlchemy.IsNotEmpty(q.Field("external_id"))))
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()

		zoneQ := wires.Query(sqlchemy.DISTINCT("zone_id", wires.Field("zone_id")))
		zoneQ = zoneQ.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		zoneQ = zoneQ.Filter(sqlchemy.Equals(networks.Field("status"), NETWORK_STATUS_AVAILABLE))

		q = q.Filter(sqlchemy.In(q.Field("id"), zoneQ.SubQuery()))
	}
	return q, nil
}

func (self *SZone) AllowGetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SZone) GetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetCapabilities(ctx, userCred, query, self)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SZone) isManaged() bool {
	region := self.GetRegion()
	if region != nil && len(region.ExternalId) == 0 {
		return false
	} else {
		return true
	}
}

func (self *SZone) isSchedPolicySupported() bool {
	return !self.isManaged()
}

func (self *SZone) getMinNicCount() int {
	return 1
}

func (self *SZone) getMaxNicCount() int {
	if self.isManaged() {
		return 1
	} else {
		return 8
	}
}

func (self *SZone) getMinDataDiskCount() int {
	return 0
}

func (self *SZone) getMaxDataDiskCount() int {
	return 6
}

func (manager *SZoneManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionStr := jsonutils.GetAnyString(query, []string{"region", "region_id", "cloudregion", "cloudregion_id"})
	var regionId string
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
			} else {
				return nil, httperrors.NewInternalServerError("Query region %s fail %s", regionStr, err)
			}
		}
		regionId = regionObj.GetId()
	} else {
		regionId = "default"
	}
	data.Add(jsonutils.NewString(regionId), "cloudregion_id")
	return manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}
