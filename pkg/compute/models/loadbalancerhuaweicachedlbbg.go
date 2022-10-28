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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
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

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	AssociatedId   string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`  // 关联ID
	AssociatedType string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`  // 关联类型， listener || rule
	ProtocolType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 监听协议类型
}

func (manager *SHuaweiCachedLbbgManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
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
	err := HuaweiCachedLbManager.Query().Equals("cached_backend_group_id", lbbg.GetId()).IsFalse("pending_deleted").All(&ret)
	if err != nil {
		log.Errorf("failed to get cached backends for backendgroup %s", lbbg.Name)
		return nil, err
	}

	return ret, nil
}

func (lbbg *SHuaweiCachedLbbg) GetICloudLoadbalancerBackendGroup(ctx context.Context) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if len(lbbg.ExternalId) == 0 {
		return nil, fmt.Errorf("backendgroup %s has no external id", lbbg.GetId())
	}

	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("backendgroup %s releated loadbalancer not found", lbbg.GetId())
	}

	iregion, err := lb.GetIRegion(ctx)
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
	err := man.Query().IsFalse("pending_deleted").Equals("backend_group_id", backendGroupId).Equals("protocol_type", protocolType).IsNullOrEmpty("associated_id").IsNotEmpty("external_id").All(&ret)
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
	err := man.Query().IsFalse("pending_deleted").Equals("associated_id", associateId).First(ret)
	if err != nil {
		return nil, err
	}

	ret.SetModelManager(man, ret)
	return ret, nil
}

func (man *SHuaweiCachedLbbgManager) GetCachedBackendGroups(backendGroupId string) ([]SHuaweiCachedLbbg, error) {
	ret := []SHuaweiCachedLbbg{}
	err := man.Query().IsFalse("pending_deleted").Equals("backend_group_id", backendGroupId).All(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SHuaweiCachedLbbgManager) getLoadbalancerBackendgroupsByLoadbalancer(lb *SLoadbalancer) ([]SHuaweiCachedLbbg, error) {
	lbbgs := []SHuaweiCachedLbbg{}
	q := man.Query().IsFalse("pending_deleted").Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for lb: %s error: %v", lb.Name, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SHuaweiCachedLbbgManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SHuaweiCachedLbbg, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backendgroups", fmt.Sprintf("%s-%s", provider.Id, lb.Id))
	defer lockman.ReleaseRawObject(ctx, "backendgroups", fmt.Sprintf("%s-%s", provider.Id, lb.Id))

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
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i], provider.GetOwnerId(), provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
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

	err := lbbg.ValidateDeleteCondition(ctx, nil)
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

func (lbbg *SHuaweiCachedLbbg) isBackendsMatch(backends []SLoadbalancerBackend, ibackends []cloudprovider.ICloudLoadbalancerBackend) bool {
	if len(ibackends) != len(backends) {
		return false
	}

	locals := []string{}
	remotes := []string{}

	for i := range backends {
		guest := backends[i].GetGuest()
		seg := strings.Join([]string{guest.ExternalId, strconv.Itoa(backends[i].Weight), strconv.Itoa(backends[i].Port)}, "/")
		locals = append(locals, seg)
	}

	for i := range ibackends {
		ibackend := ibackends[i]
		seg := strings.Join([]string{ibackend.GetBackendId(), strconv.Itoa(ibackend.GetWeight()), strconv.Itoa(ibackend.GetPort())}, "/")
		remotes = append(remotes, seg)
	}

	for i := range remotes {
		if !utils.IsInStringArray(remotes[i], locals) {
			return false
		}
	}

	return true
}

func (lbbg *SHuaweiCachedLbbg) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	lbbg.SetModelManager(HuaweiCachedLbbgManager, lbbg)

	ibackends, err := extLoadbalancerBackendgroup.GetILoadbalancerBackends()
	if err != nil {
		return errors.Wrap(err, "HuaweiCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetILoadbalancerBackends")
	}

	localLbbg, err := lbbg.GetLocalBackendGroup(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "HuaweiCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetLocalBackendGroup")
	}

	backends, err := localLbbg.GetBackends()
	if err != nil {
		return errors.Wrap(err, "HuaweiCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.GetBackends")
	}

	var newLocalLbbg *SLoadbalancerBackendGroup
	if !lbbg.isBackendsMatch(backends, ibackends) {
		newLocalLbbg, err = newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId, provider)
		if err != nil {
			return errors.Wrap(err, "HuaweiCachedLbbg.SyncWithCloudLoadbalancerBackendgroup.newLocalBackendgroupFromCloudLoadbalancerBackendgroup")
		}
	}

	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
		if newLocalLbbg != nil {
			lbbg.BackendGroupId = newLocalLbbg.GetId()
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)
	return err
}

func (man *SHuaweiCachedLbbgManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SHuaweiCachedLbbg, error) {
	LocalLbbg, err := newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, extLoadbalancerBackendgroup, syncOwnerId, provider)
	if err != nil {
		return nil, err
	}

	lbbg := &SHuaweiCachedLbbg{}
	lbbg.SetModelManager(man, lbbg)

	region, err := lb.GetRegion()
	if err != nil {
		return nil, err
	}

	lbbg.ManagerId = provider.Id
	lbbg.CloudregionId = region.Id
	lbbg.LoadbalancerId = lb.Id
	lbbg.BackendGroupId = LocalLbbg.GetId()
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()
	lbbg.ProtocolType = extLoadbalancerBackendgroup.GetProtocolType()

	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, LocalLbbg.GetName())
		if err != nil {
			return err
		}
		lbbg.Name = newName

		return man.TableSpec().Insert(ctx, lbbg)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)
	return lbbg, nil
}

func newLocalBackendgroupFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SLoadbalancerBackendGroup, error) {
	localman := LoadbalancerBackendGroupManager
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(localman, lbbg)

	lbbg.LoadbalancerId = lb.Id
	// lbbg.CloudregionId = lb.CloudregionId
	// lbbg.ManagerId = lb.ManagerId
	lbbg.ExternalId = ""

	lbbg.Type = extLoadbalancerBackendgroup.GetType()
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	var err = func() error {
		lockman.LockRawObject(ctx, localman.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, localman.Keyword(), "name")

		newName, err := db.GenerateName(ctx, localman, syncOwnerId, extLoadbalancerBackendgroup.GetName())
		if err != nil {
			return err
		}
		lbbg.Name = newName

		return localman.TableSpec().Insert(ctx, lbbg)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)

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
