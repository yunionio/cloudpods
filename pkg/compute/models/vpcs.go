package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	VPC_STATUS_PENDING       = "pending"
	VPC_STATUS_AVAILABLE     = "available"
	VPC_STATUS_FAILED        = "failed"
	VPC_STATUS_START_DELETE  = "start_delete"
	VPC_STATUS_DELETING      = "deleting"
	VPC_STATUS_DELETE_FAILED = "delete_failed"
	VPC_STATUS_DELETED       = "deleted"
	VPC_STATUS_UNKNOWN       = "unknown"

	MAX_VPC_PER_REGION = 3
)

type SVpcManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var VpcManager *SVpcManager

func init() {
	VpcManager = &SVpcManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SVpc{},
			"vpcs_tbl",
			"vpc",
			"vpcs",
		),
	}
}

type SVpc struct {
	db.SEnabledStatusStandaloneResourceBase
	SManagedResourceBase

	IsDefault bool `default:"false" list:"admin" create:"admin_optional"`

	CidrBlock string `width:"64" charset:"ascii" nullable:"true" list:"admin" create:"admin_required"`

	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
}

func (manager *SVpcManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{CloudregionManager}
}

func (self *SVpcManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SVpcManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SVpc) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SVpc) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SVpc) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
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
	if self.GetNetworkCount() > 0 {
		return httperrors.NewNotEmptyError("VPC not empty")
	}
	if self.Id == "default" {
		return httperrors.NewProtectedResourceError("not allow to delete default vpc")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SVpc) getWireQuery() *sqlchemy.SQuery {
	wires := WireManager.Query()
	if self.Id == "default" {
		return wires.Filter(sqlchemy.OR(sqlchemy.IsNull(wires.Field("vpc_id")),
			sqlchemy.IsEmpty(wires.Field("vpc_id")),
			sqlchemy.Equals(wires.Field("vpc_id"), self.Id)))
	} else {
		return wires.Equals("vpc_id", self.Id)
	}
}

func (self *SVpc) GetWireCount() int {
	q := self.getWireQuery()
	return q.Count()
}

func (self *SVpc) GetWires() []SWire {
	wires := make([]SWire, 0)
	q := self.getWireQuery()
	err := db.FetchModelObjects(WireManager, q, &wires)
	if err != nil {
		log.Errorf("getWires fail %s", err)
		return nil
	}
	return wires
}

func (self *SVpc) getNetworkQuery() *sqlchemy.SQuery {
	q := NetworkManager.Query()
	wireQ := self.getWireQuery().SubQuery()
	q = q.In("wire_id", wireQ.Query(wireQ.Field("id")).SubQuery())
	return q
}

func (self *SVpc) GetNetworkCount() int {
	q := self.getNetworkQuery()
	return q.Count()
}

func (self *SVpc) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.GetWireCount())), "wire_count")
	extra.Add(jsonutils.NewInt(int64(self.GetNetworkCount())), "network_count")
	region := self.GetRegion()
	if region != nil {
		extra.Add(jsonutils.NewString(region.GetName()), "region")
		if len(region.GetExternalId()) > 0 {
			extra.Add(jsonutils.NewString(region.GetExternalId()), "region_external_id")
		}
	}
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

func (manager *SVpcManager) getVpcsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
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

func (manager *SVpcManager) SyncVPCs(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, vpcs []cloudprovider.ICloudVpc) ([]SVpc, []cloudprovider.ICloudVpc, compare.SyncResult) {
	localVPCs := make([]SVpc, 0)
	remoteVPCs := make([]cloudprovider.ICloudVpc, 0)
	syncResult := compare.SyncResult{}

	dbVPCs, err := manager.getVpcsByRegion(region, provider)
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
		// err = removed[i].ValidateDeleteCondition(ctx)
		// if err != nil { // cannot delete
		removed[i].markAllNetworksUnknown(userCred)
		_, err = removed[i].PerformDisable(ctx, userCred, nil, nil)
		if err == nil {
			err = removed[i].SetStatus(userCred, VPC_STATUS_UNKNOWN, "sync to delete")
		}
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
		// } else {
		// 	err = removed[i].Delete(ctx, userCred)
		// 	if err != nil {
		//		syncResult.DeleteError(err)
		//	} else {
		//		syncResult.Delete()
		//	}
		// }
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudVpc(commonext[i])
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

func (self *SVpc) SyncWithCloudVpc(extVPC cloudprovider.ICloudVpc) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		extVPC.Refresh()
		self.Name = extVPC.GetName()
		self.Status = extVPC.GetStatus()
		self.CidrBlock = extVPC.GetCidrBlock()
		self.IsDefault = extVPC.GetIsDefault()
		self.ExternalId = extVPC.GetGlobalId()

		self.IsEmulated = extVPC.IsEmulated()

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

	vpc.ManagerId = extVPC.GetManagerId()

	vpc.IsEmulated = extVPC.IsEmulated()

	err := manager.TableSpec().Insert(&vpc)
	if err != nil {
		log.Errorf("newFromCloudVpc fail %s", err)
		return nil, err
	}
	return &vpc, nil
}

func (self *SVpc) markAllNetworksUnknown(userCred mcclient.TokenCredential) error {
	wires := self.GetWires()
	if wires == nil || len(wires) == 0 {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wires[i].markNetworkUnknown(userCred)
	}
	return nil
}

func (manager *SVpcManager) InitializeData() error {
	vpcObj, err := manager.FetchById("default")
	if err != nil {
		if err == sql.ErrNoRows {
			defVpc := SVpc{}
			defVpc.SetModelManager(VpcManager)

			defVpc.Id = "default"
			defVpc.Name = "Default"
			defVpc.CloudregionId = "default"
			defVpc.Description = "Default VPC"
			defVpc.Status = VPC_STATUS_AVAILABLE
			defVpc.IsDefault = true
			err = manager.TableSpec().Insert(&defVpc)
			if err != nil {
				log.Errorf("Insert default vpc fail: %s", err)
			}
			return err
		} else {
			return err
		}
	} else {
		vpc := vpcObj.(*SVpc)
		if vpc.Status != VPC_STATUS_AVAILABLE {
			_, err = manager.TableSpec().Update(vpc, func() error {
				vpc.Status = VPC_STATUS_AVAILABLE
				return nil
			})
			return err
		}
	}
	return nil
}

func (manager *SVpcManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionId, err := data.GetString("cloudregion_id")
	if err != nil {
		return nil, httperrors.NewInputParameterError("No cloudregion_id")
	}
	region := CloudregionManager.FetchRegionById(regionId)
	if region == nil {
		return nil, httperrors.NewInputParameterError("Invalid cloudregion_id")
	}
	if region.isManaged() {
		managerStr, _ := data.GetString("manager_id")
		if len(managerStr) == 0 {
			managerStr, _ = data.GetString("manager")
			if len(managerStr) == 0 {
				return nil, httperrors.NewInputParameterError("cloud provider/manager must be provided")
			}
		}
		managerObj := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Cloud provider/manager %s not found", managerStr)
		}
		data.Add(jsonutils.NewString(managerObj.GetId()), "manager_id")
	} else {
		return nil, httperrors.NewNotImplementedError("Cannot create VPC in private cloud")
	}

	cidrBlock, _ := data.GetString("cidr_block")
	if len(cidrBlock) > 0 {
		_, err = netutils.NewIPV4Prefix(cidrBlock)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid cidr_block %s", cidrBlock)
		}
	}
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SVpc) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if len(self.ManagerId) == 0 {
		return
	}
	task, err := taskman.TaskManager.NewTask(ctx, "VpcCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("VpcCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (self *SVpc) GetIRegion() (cloudprovider.ICloudRegion, error) {
	region := self.GetRegion()
	if region == nil {
		log.Errorf("cannot find region for this vpc??")
		return nil, fmt.Errorf("Cannot find region")
	}
	provider, err := self.GetDriver()
	if err != nil {
		log.Errorf("fail to find cloud provider")
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SVpc) GetIVpc() (cloudprovider.ICloudVpc, error) {
	provider, err := self.GetDriver()
	if err != nil {
		log.Errorf("fail to find cloud provider")
		return nil, err
	}
	return provider.GetIVpcById(self.GetExternalId())
}

func (self *SVpc) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SVpc delete do nothing")
	self.SetStatus(userCred, VPC_STATUS_START_DELETE, "")
	return nil
}

func (self *SVpc) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.ExternalId) > 0 {
		return self.StartDeleteVpcTask(ctx, userCred)
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SVpc) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(), userCred)
	self.SetStatus(userCred, VPC_STATUS_DELETED, "real delete")
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SVpc) StartDeleteVpcTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "VpcDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("Start vpcdeleteTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVpc) getPrefix() netutils.IPV4Prefix {
	if len(self.CidrBlock) > 0 {
		prefix, _ := netutils.NewIPV4Prefix(self.CidrBlock)
		return prefix
	}
	return netutils.IPV4Prefix{}
}

func (self *SVpc) getIPRange() netutils.IPV4AddrRange {
	pref := self.getPrefix()
	return pref.ToIPRange()
}

func (self *SVpc) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SVpc) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	provider := self.GetCloudprovider()
	if provider != nil {
		if provider.Enabled {
			return nil, httperrors.NewInvalidStatusError("Cannot purge vpc on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (manager *SVpcManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "provider", "manager_id", "provider_id"})
	if len(managerStr) > 0 {
		provider := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if provider == nil {
			return nil, httperrors.NewResourceNotFoundError("provider %s not found", managerStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
	}

	return q, nil
}
