package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHuaweiCachedLbbgManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var HuaweiCachedLbbgManager *SHuaweiCachedLbbgManager

func init() {
	HuaweiCachedLbbgManager = &SHuaweiCachedLbbgManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SHuaweiCachedLbbg{},
			"huaweicachedlbbgs_tbl",
			"huaweicachedlbbg",
			"huaweicachedlbbgs",
		),
	}
	HuaweiCachedLbbgManager.SetVirtualObject(HuaweiCachedLbbgManager)
}

type SHuaweiCachedLbbg struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	AssociatedId   string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"` // 关联ID
	AssociatedType string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"` // 关联类型， listener || rule
	ProtocolType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 监听协议类型
}

func (lbb *SHuaweiCachedLbbg) GetCustomizeColumns(context.Context, mcclient.TokenCredential, jsonutils.JSONObject) *jsonutils.JSONDict {
	return nil
}

func (lbbg *SHuaweiCachedLbbg) GetLocalBackendGroup(ctx context.Context, userCred mcclient.TokenCredential) (*SLoadbalancerBackendGroup, error) {
	if len(lbbg.BackendGroupId) == 0 {
		return nil, fmt.Errorf("GetLocalBackendGroup no related local backendgroup")
	}

	locallbbg, err := db.FetchById(LoadbalancerBackendGroupManager, lbbg.BackendGroupId)
	if err != nil {
		return nil, err
	}

	return locallbbg.(*SLoadbalancerBackendGroup), err
}

func (lbbg *SHuaweiCachedLbbg) GetLoadbalancer() *SLoadbalancer {
	lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	if err != nil {
		log.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		return nil
	}
	return lb.(*SLoadbalancer)
}

func (lbbg *SHuaweiCachedLbbg) GetCachedBackends() ([]SHuaweiCachedLb, error) {
	ret := []SHuaweiCachedLb{}
	err := HuaweiCachedLbManager.TableSpec().Query().Equals("cached_backend_group_id", lbbg.GetId()).All(&ret)
	if err != nil {
		log.Errorf("failed to get cached backends for backendgroup %s", lbbg.Name)
		return nil, err
	}

	return ret, nil
}

func (lbbg *SHuaweiCachedLbbg) GetICloudLoadbalancerBackendGroup() (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if len(lbbg.ExternalId) == 0 {
		return nil, fmt.Errorf("backendgroup %s has no external id", lbbg.GetId())
	}

	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("backendgroup %s releated loadbalancer not found", lbbg.GetId())
	}

	iregion, err := lb.GetIRegion()
	if err != nil {
		return nil, err
	}

	ilb, err := iregion.GetILoadBalancerById(lb.GetExternalId())
	if err != nil {
		return nil, err
	}

	ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
	if err != nil {
		return nil, err
	}

	return ilbbg, nil
}

func (man *SHuaweiCachedLbbgManager) GetUsableCachedBackendGroups(backendGroupId string, protocolType string) ([]SHuaweiCachedLbbg, error) {
	ret := []SHuaweiCachedLbbg{}
	err := man.TableSpec().Query().Equals("backend_group_id", backendGroupId).Equals("protocol_type", protocolType).IsNullOrEmpty("associated_id").IsNotEmpty("external_id").All(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func (man *SHuaweiCachedLbbgManager) GetUsableCachedBackendGroup(backendGroupId string, protocolType string) (*SHuaweiCachedLbbg, error) {
	ret, err := man.GetUsableCachedBackendGroups(backendGroupId, protocolType)
	if err != nil {
		return nil, err
	}

	if len(ret) > 0 {
		return &ret[0], nil
	}

	return nil, nil
}

func (man *SHuaweiCachedLbbgManager) GetCachedBackendGroupByAssociateId(associateId string) (*SHuaweiCachedLbbg, error) {
	ret := &SHuaweiCachedLbbg{}
	err := man.TableSpec().Query().Equals("associated_id", associateId).First(ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SHuaweiCachedLbbgManager) GetCachedBackendGroups(backendGroupId string) ([]SHuaweiCachedLbbg, error) {
	ret := []SHuaweiCachedLbbg{}
	err := man.TableSpec().Query().Equals("backend_group_id", backendGroupId).All(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SHuaweiCachedLbbgManager) getLoadbalancerBackendgroupsByLoadbalancer(lb *SLoadbalancer) ([]SHuaweiCachedLbbg, error) {
	lbbgs := []SHuaweiCachedLbbg{}
	q := man.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for lb: %s error: %v", lb.Name, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SHuaweiCachedLbbgManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SHuaweiCachedLbbg, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

	localLbgs := []SHuaweiCachedLbbg{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByLoadbalancer(lb)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SHuaweiCachedLbbg{}
	commondb := []SHuaweiCachedLbbg{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendGroup{}

	err = compare.CompareSets(dbLbbgs, lbbgs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackendgroup(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i], provider.GetOwnerId())
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localLbgs = append(localLbgs, *new)
			remoteLbbgs = append(remoteLbbgs, added[i])
			syncResult.Add()
		}
	}
	return localLbgs, remoteLbbgs, syncResult
}

func (lbbg *SHuaweiCachedLbbg) syncRemoveCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbbg.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lbbg.SetModelManager(HuaweiCachedLbbgManager, lbbg)
		err := db.DeleteModel(ctx, userCred, lbbg)
		if err != nil {
			return err
		}
	}
	return err
}

func (lbbg *SHuaweiCachedLbbg) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider) error {
	lbbg.SetModelManager(HuaweiCachedLbbgManager, lbbg)
	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, lb.ManagerId)
	return err
}

func (man *SHuaweiCachedLbbgManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider) (*SHuaweiCachedLbbg, error) {
	LocalLbbg, err := newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId)
	if err != nil {
		return nil, err
	}

	lbbg := &SHuaweiCachedLbbg{}
	lbbg.SetModelManager(man, lbbg)

	lbbg.ManagerId = lb.ManagerId
	lbbg.CloudregionId = lb.CloudregionId
	lbbg.LoadbalancerId = lb.Id
	lbbg.BackendGroupId = LocalLbbg.GetId()
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()
	lbbg.ProtocolType = extLoadbalancerBackendgroup.GetProtocolType()

	newName, err := db.GenerateName(man, syncOwnerId, LocalLbbg.GetName())
	if err != nil {
		return nil, err
	}

	lbbg.Name = newName
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = man.TableSpec().Insert(lbbg)
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, lb.ManagerId)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)
	return lbbg, nil
}

func newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancerBackendGroup, error) {
	localman := LoadbalancerBackendGroupManager
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(localman, lbbg)

	lbbg.LoadbalancerId = lb.Id
	lbbg.CloudregionId = lb.CloudregionId
	lbbg.ManagerId = lb.ManagerId
	lbbg.ExternalId = ""

	newName, err := db.GenerateName(localman, syncOwnerId, extLoadbalancerBackendgroup.GetName())
	if err != nil {
		return nil, err
	}

	lbbg.Name = newName
	lbbg.Type = extLoadbalancerBackendgroup.GetType()
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = localman.TableSpec().Insert(lbbg)
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, lb.ManagerId)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)

	if extLoadbalancerBackendgroup.IsDefault() {
		lb.SetModelManager(localman, lb)
		_, err := db.Update(lb, func() error {
			lb.BackendGroupId = lbbg.Id
			return nil
		})
		if err != nil {
			log.Errorf("failed to set backendgroup id for lb %s error: %v", lb.Name, err)
		}
	}
	return lbbg, nil
}
