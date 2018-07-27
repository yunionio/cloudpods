package models

import (
	"context"
	"database/sql"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/pkg/util/compare"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
)

const (
	VPC_STATUS_PENDING   = "pending"
	VPC_STATUS_AVAILABLE = "available"
)

type SVpcManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var VpcManager *SVpcManager

func init() {
	VpcManager = &SVpcManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SVpc{}, "vpcs_tbl", "vpc", "vpcs")}
}

type SVpc struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure

	IsDefault     bool   `default:"false" list:"admin" create:"admin_required"`
	CidrBlock     string `width:"64" charset:"ascii" nullable:"true" list:"admin" create:"admin_required"`
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
}

func (manager *SVpcManager) GetContextManager() db.IModelManager {
	return CloudregionManager
}

func (self *SVpc) GetCloudRegionId() string {
	if len(self.CloudregionId) == 0 {
		return "default"
	} else {
		return self.CloudregionId
	}
}

func (self *SVpc) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		self.Id = idstr
	}
	return nil
}

func (self *SVpc) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetWireCount() > 0 {
		return httperrors.NewNotEmptyError("VPC not empty")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SVpc) GetWireCount() int {
	wires := WireManager.Query()
	if self.Id == "default" {
		return wires.Filter(sqlchemy.OR(sqlchemy.IsNull(wires.Field("vpc_id")),
			sqlchemy.IsEmpty(wires.Field("vpc_id")),
			sqlchemy.Equals(wires.Field("vpc_id"), self.Id))).Count()
	} else {
		return wires.Equals("vpc_id", self.Id).Count()
	}
}

func (self *SVpc) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.GetWireCount())), "wire_count")
	region := self.GetRegion()
	extra.Add(jsonutils.NewString(region.GetName()), "region")
	return extra
}

func (self *SVpc) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		log.Errorf("Get region error %s", err)
		return nil
	}
	return region.(*SCloudregion)
}

func (self *SVpc) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SVpc) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (manager *SVpcManager) getVpcsByRegion(region *SCloudregion) ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	err := db.FetchModelObjects(manager, q, &vpcs)
	if err != nil {
		return nil, err
	}
	return vpcs, nil
}

func (self *SVpc) setDefault(def bool) error {
	var err error
	if self.IsDefault != def {
		_, err = self.GetModelManager().TableSpec().Update(self, func() error {
			self.IsDefault = def
			return nil
		})
	}
	return err
}

func (manager *SVpcManager) SyncVPCs(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, vpcs []cloudprovider.ICloudVpc) ([]SVpc, []cloudprovider.ICloudVpc, compare.SyncResult) {
	localVPCs := make([]SVpc, 0)
	remoteVPCs := make([]cloudprovider.ICloudVpc, 0)
	syncResult := compare.SyncResult{}

	dbVPCs, err := manager.getVpcsByRegion(region)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SVpc, 0)
	commondb := make([]SVpc, 0)
	commonext := make([]cloudprovider.ICloudVpc, 0)
	added := make([]cloudprovider.ICloudVpc, 0)

	err = compare.CompareSets(dbVPCs, vpcs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			_, err = removed[i].PerformDisable(ctx, userCred, nil, nil)
			if err == nil {
				err = removed[i].SetStatus(userCred, VPC_STATUS_PENDING, "sync to delete")
			}
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
		err = commondb[i].syncWithCloudVpc(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localVPCs = append(localVPCs, commondb[i])
			remoteVPCs = append(remoteVPCs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudVpc(added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localVPCs = append(localVPCs, *new)
			remoteVPCs = append(remoteVPCs, added[i])
			syncResult.Add()
		}
	}

	return localVPCs, remoteVPCs, syncResult
}

func (self *SVpc) syncWithCloudVpc(extVPC cloudprovider.ICloudVpc) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = extVPC.GetName()
		self.Status = extVPC.GetStatus()
		self.CidrBlock = extVPC.GetCidrBlock()
		self.IsDefault = extVPC.GetIsDefault()
		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudVpc error %s", err)
	}
	return err
}

func (manager *SVpcManager) newFromCloudVpc(extVPC cloudprovider.ICloudVpc, region *SCloudregion) (*SVpc, error) {
	vpc := SVpc{}
	vpc.SetModelManager(manager)

	vpc.Name = extVPC.GetName()
	vpc.Status = extVPC.GetStatus()
	vpc.ExternalId = extVPC.GetGlobalId()
	vpc.IsDefault = extVPC.GetIsDefault()
	vpc.CidrBlock = extVPC.GetCidrBlock()
	vpc.CloudregionId = region.Id

	err := manager.TableSpec().Insert(&vpc)
	if err != nil {
		log.Errorf("newFromCloudVpc fail %s", err)
		return nil, err
	}
	return &vpc, nil
}

func (manager *SVpcManager) InitializeData() error {
	_, err := manager.FetchById("default")
	if err != nil {
		if err == sql.ErrNoRows {
			defVpc := SVpc{}
			defVpc.Id = "default"
			defVpc.Name = "Default"
			defVpc.CloudregionId = "default"
			defVpc.Description = "Default VPC"
			defVpc.IsDefault = true
			err = manager.TableSpec().Insert(&defVpc)
			if err != nil {
				log.Errorf("Insert default vpc fail: %s", err)
			}
			return err
		} else {
			return err
		}
	}
	return nil
}
