package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_REGION_STATUS_INSERVER     = "inservice"
	CLOUD_REGION_STATUS_OUTOFSERVICE = "outofservice"
)

type SCloudregionManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var CloudregionManager *SCloudregionManager

func init() {
	CloudregionManager = &SCloudregionManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SCloudregion{}, "cloudregions_tbl", "cloudregion", "cloudregions")}
}

type SCloudregion struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure

	Latitude  float32 `list:"user"`
	Longitude float32 `list:"user"`
	Provider  string  `width:"64" charset:"ascii" list:"user"`
}

func (manager *SCloudregionManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SCloudregion) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		self.Id = idstr
	}
	return nil
}

func (self *SCloudregion) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetZoneCount() > 0 || self.GetVpcCount() > 0 {
		return httperrors.NewNotEmptyError("not empty cloud region")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudregion) GetZoneCount() int {
	zones := ZoneManager.Query()
	if self.Id == "default" {
		return zones.Filter(sqlchemy.OR(sqlchemy.IsNull(zones.Field("cloudregion_id")),
			sqlchemy.IsEmpty(zones.Field("cloudregion_id")),
			sqlchemy.Equals(zones.Field("cloudregion_id"), self.Id))).Count()
	} else {
		return zones.Equals("cloudregion_id", self.Id).Count()
	}
}

func (self *SCloudregion) GetVpcCount() int {
	vpcs := VpcManager.Query()
	if self.Id == "default" {
		return vpcs.Filter(sqlchemy.OR(sqlchemy.IsNull(vpcs.Field("cloudregion_id")),
			sqlchemy.IsEmpty(vpcs.Field("cloudregion_id")),
			sqlchemy.Equals(vpcs.Field("cloudregion_id"), self.Id))).Count()
	} else {
		return vpcs.Equals("cloudregion_id", self.Id).Count()
	}
}

func (self *SCloudregion) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.GetVpcCount())), "vpc_count")
	extra.Add(jsonutils.NewInt(int64(self.GetZoneCount())), "zone_count")
	return extra
}

func (self *SCloudregion) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SCloudregion) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (manager *SCloudregionManager) getRegionByProvider(provider string) ([]SCloudregion, error) {
	regions := make([]SCloudregion, 0)
	q := manager.Query().Startswith("external_id", provider)
	err := db.FetchModelObjects(manager, q, &regions)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return regions, nil
}

func (manager *SCloudregionManager) SyncRegions(ctx context.Context, userCred mcclient.TokenCredential, provider string, regions []cloudprovider.ICloudRegion) ([]SCloudregion, []cloudprovider.ICloudRegion, compare.SyncResult) {
	syncResult := compare.SyncResult{}
	localRegions := make([]SCloudregion, 0)
	remoteRegions := make([]cloudprovider.ICloudRegion, 0)

	dbRegions, err := manager.getRegionByProvider(provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}
	log.Debugf("Region with provider %s %d", provider, len(dbRegions))

	removed := make([]SCloudregion, 0)
	commondb := make([]SCloudregion, 0)
	commonext := make([]cloudprovider.ICloudRegion, 0)
	added := make([]cloudprovider.ICloudRegion, 0)
	err = compare.CompareSets(dbRegions, regions, &removed, &commondb, &commonext, &added)
	if err != nil {
		log.Errorf("compare regions fail %s", err)
		syncResult.Error(err)
		return nil, nil, syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err == nil {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].SetStatus(userCred, CLOUD_REGION_STATUS_OUTOFSERVICE, "Out of sync")
			if err == nil {
				_, err = removed[i].PerformDisable(ctx, userCred, nil, nil)
			}
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		// update
		err = commondb[i].syncWithCloudRegion(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localRegions = append(localRegions, commondb[i])
			remoteRegions = append(remoteRegions, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudRegion(added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			localRegions = append(localRegions, *new)
			remoteRegions = append(remoteRegions, added[i])
			syncResult.Add()
		}
	}
	return localRegions, remoteRegions, syncResult
}

func (self *SCloudregion) syncWithCloudRegion(cloudRegion cloudprovider.ICloudRegion) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = cloudRegion.GetName()
		self.Status = cloudRegion.GetStatus()
		self.Latitude = cloudRegion.GetLatitude()
		self.Longitude = cloudRegion.GetLongitude()
		self.Provider = cloudRegion.GetProvider()

		self.IsEmulated = cloudRegion.IsEmulated()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudRegion %s", err)
	}
	return err
}

func (manager *SCloudregionManager) newFromCloudRegion(cloudRegion cloudprovider.ICloudRegion) (*SCloudregion, error) {
	region := SCloudregion{}
	region.SetModelManager(manager)

	region.ExternalId = cloudRegion.GetGlobalId()
	region.Name = cloudRegion.GetName()
	region.Latitude = cloudRegion.GetLatitude()
	region.Longitude = cloudRegion.GetLongitude()
	region.Status = cloudRegion.GetStatus()
	region.Enabled = true
	region.Provider = cloudRegion.GetProvider()

	region.IsEmulated = cloudRegion.IsEmulated()

	err := manager.TableSpec().Insert(&region)
	if err != nil {
		log.Errorf("newFromCloudRegion fail %s", err)
		return nil, err
	}
	return &region, nil
}

func (self *SCloudregion) AllowPerformDefaultVpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCloudregion) PerformDefaultVpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	vpcs, err := VpcManager.getVpcsByRegion(self)
	if err != nil {
		return nil, err
	}
	vpcStr, _ := data.GetString("vpc")
	if len(vpcStr) == 0 {
		return nil, httperrors.NewInputParameterError("no vpc id")
	}
	findVpc := false
	for _, vpc := range vpcs {
		if vpc.Id == vpcStr || vpc.Name == vpcStr {
			findVpc = true
			break
		}
	}
	if !findVpc {
		return nil, httperrors.NewResourceNotFoundError("VPC %s not found", vpcStr)
	}
	for _, vpc := range vpcs {
		if vpc.Id == vpcStr || vpc.Name == vpcStr {
			err = vpc.setDefault(true)
		} else {
			err = vpc.setDefault(false)
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (manager *SCloudregionManager) FetchRegionById(id string) *SCloudregion {
	obj, err := manager.FetchById(id)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SCloudregion)
}

func (manager *SCloudregionManager) InitializeData() error {
	// check if default region exists
	_, err := manager.FetchById("default")
	if err != nil {
		if err == sql.ErrNoRows {
			defRegion := SCloudregion{}
			defRegion.Id = "default"
			defRegion.Name = "Default"
			defRegion.Enabled = true
			defRegion.Description = "Default Region"
			defRegion.Status = CLOUD_REGION_STATUS_INSERVER
			err = manager.TableSpec().Insert(&defRegion)
			if err != nil {
				log.Errorf("Insert default region fails: %s", err)
			}
			return err
		} else {
			return err
		}
	}
	return nil
}

func (manager *SCloudregionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
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
	managerStr, _ := query.GetString("manager")
	if len(managerStr) > 0 {
		manager := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if manager == nil {
			return nil, httperrors.NewResourceNotFoundError("Cloud provider/manager %s not found", managerStr)
		}
		q = q.Equals("provider", manager.Provider)
	}
	if jsonutils.QueryBoolean(query, "usable", false) {
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()

		sq := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
		sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), NETWORK_STATUS_AVAILABLE))

		q = q.Filter(sqlchemy.In(q.Field("id"), sq.SubQuery()))
	}
	return q, nil
}

func (manager *SCloudregionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SCloudregion) isManaged() bool {
	if len(self.ExternalId) > 0 {
		return true
	} else {
		return false
	}
}

func (self *SCloudregion) ValidateUpdateCondition(ctx context.Context) error {
	if len(self.ExternalId) > 0 {
		return httperrors.NewConflictError("Cannot update external resource")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}
