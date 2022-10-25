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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var LB_CERTS_TO_BE_PURGE = map[string][]string{}

type IPurgeableManager interface {
	Keyword() string
	purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error
}

func (eipManager *SElasticipManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	eips := make([]SElasticip, 0)
	err := fetchByManagerId(eipManager, providerId, &eips)
	if err != nil {
		return err
	}
	for i := range eips {
		err := eips[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (eip *SElasticip) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, eip)
	defer lockman.ReleaseObject(ctx, eip)

	err := eip.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return eip.RealDelete(ctx, userCred)
}

func (hostManager *SHostManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	hosts := make([]SHost, 0)
	err := fetchByManagerId(hostManager, providerId, &hosts)
	if err != nil {
		return err
	}
	for i := range hosts {
		err := hosts[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (host *SHost) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)

	_, err := host.PerformDisable(ctx, userCred, nil, apis.PerformDisableInput{})
	if err != nil {
		return errors.Wrapf(err, "PerformDisable")
	}

	guests, err := host.GetGuests()
	if err != nil {
		return errors.Wrapf(err, "host.GetGuests")
	}
	for i := range guests {
		err := guests[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "purge guest %s", guests[i].Id)
		}
	}

	// clean all disks on locally attached storages
	storages := host._getAttachedStorages(tristate.None, tristate.None, api.HOST_STORAGE_LOCAL_TYPES)
	for i := range storages {
		err := storages[i].purgeDisks(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "purgeDisks for storage %s", storages[i].Name)
		}
	}

	err = host.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	return host.RealDelete(ctx, userCred)
}

func (guest *SGuest) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	guest.SetDisableDelete(userCred, false)

	IsolatedDeviceManager.ReleaseDevicesOfGuest(ctx, guest, userCred)
	guest.RevokeAllSecgroups(ctx, userCred)
	guest.LeaveAllGroups(ctx, userCred)
	guest.DetachAllNetworks(ctx, userCred)
	guest.EjectAllIso(userCred)
	guest.EjectAllVfd(userCred)
	guest.DeleteEip(ctx, userCred)
	guest.purgeInstanceSnapshots(ctx, userCred)
	guest.DeleteAllDisksInDB(ctx, userCred)
	if !utils.IsInStringArray(guest.Hypervisor, HypervisorIndependentInstanceSnapshot) {
		guest.DeleteAllInstanceSnapshotInDB(ctx, userCred)
	}

	err := guest.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return guest.RealDelete(ctx, userCred)
}

func (guest *SGuest) purgeInstanceSnapshots(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	iss, err := guest.GetInstanceSnapshots()
	if err != nil {
		return errors.Wrap(err, "unable to GetInstanceSnapshots")
	}
	for i := range iss {
		err := iss[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "unable to purge InstanceSnapshot %s", iss[i].Id)
		}
	}
	return nil
}

func (is *SInstanceSnapshot) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, is)
	defer lockman.ReleaseObject(ctx, is)

	return is.RealDelete(ctx, userCred)
}

func (storage *SStorage) purgeDisks(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	disks := storage.GetDisks()
	for i := range disks {
		err := disks[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "purge disk %s", disks[i].Id)
		}
	}
	return nil
}

func (disk *SDisk) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, disk)
	defer lockman.ReleaseObject(ctx, disk)

	err := disk.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	return disk.RealDelete(ctx, userCred)
}

func (manager *SCachedLoadbalancerCertificateManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbcs := make([]SCachedLoadbalancerCertificate, 0)
	err := fetchByManagerId(manager, providerId, &lbcs)
	if err != nil {
		return err
	}

	lbcertIds := []string{}
	if certs, ok := LB_CERTS_TO_BE_PURGE[providerId]; ok {
		lbcertIds = certs
	}

	for i := range lbcs {
		err := lbcs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}

		if len(lbcs[i].CertificateId) > 0 && !utils.IsInStringArray(lbcs[i].CertificateId, lbcertIds) {
			lbcertIds = append(lbcertIds, lbcs[i].CertificateId)
		}
	}
	LB_CERTS_TO_BE_PURGE[providerId] = lbcertIds
	return nil
}

func (lbcert *SCachedLoadbalancerCertificate) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbcert)
	defer lockman.ReleaseObject(ctx, lbcert)

	err := lbcert.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	return lbcert.DoPendingDelete(ctx, userCred)
}

func (manager *SLoadbalancerCertificateManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	if certs, ok := LB_CERTS_TO_BE_PURGE[providerId]; ok {
		lbcs := make([]SLoadbalancerCertificate, 0)
		err := db.FetchModelObjects(manager, manager.Query().In("id", certs), &lbcs)
		if err != nil {
			return err
		}
		for i := range lbcs {
			err := lbcs[i].purge(ctx, userCred)
			if err != nil {
				return err
			}
		}

		delete(LB_CERTS_TO_BE_PURGE, providerId)
	}
	return nil
}

func (lbcert *SLoadbalancerCertificate) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbcert)
	defer lockman.ReleaseObject(ctx, lbcert)

	if !lbcert.PendingDeleted {
		// 内容完整的证书不需要删除
		if lbcert.IsComplete() {
			return nil
		}

		caches, err := lbcert.GetCachedCerts()
		if err != nil {
			return errors.Wrap(err, "GetCachedCerts")
		}

		if len(caches) > 0 {
			log.Debugf("the lb cert %s (%s) is in use.can not purge.", lbcert.Name, lbcert.Id)
			return nil
		}
		return lbcert.DoPendingDelete(ctx, userCred)
	}

	return nil
}

func (manager *SCachedLoadbalancerAclManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbacls := make([]SCachedLoadbalancerAcl, 0)
	err := fetchByManagerId(manager, providerId, &lbacls)
	if err != nil {
		return err
	}
	for i := range lbacls {
		err := lbacls[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbacl *SCachedLoadbalancerAcl) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbacl)
	defer lockman.ReleaseObject(ctx, lbacl)

	err := lbacl.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return lbacl.DoPendingDelete(ctx, userCred)
}

func (manager *SLoadbalancerBackendGroupManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbbgs := make([]SLoadbalancerBackendGroup, 0)
	err := fetchByLbVpcManagerId(manager, providerId, &lbbgs)
	if err != nil {
		return err
	}
	for i := range lbbgs {
		err := lbbgs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SLoadbalancerManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbs := make([]SLoadbalancer, 0)
	err := fetchByManagerId(manager, providerId, &lbs)
	if err != nil {
		return err
	}
	for i := range lbs {
		err := lbs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SLoadbalancernetworkManager) getAllLoadbalancerNetworks(lbId string) ([]SLoadbalancerNetwork, error) {
	lbnets := make([]SLoadbalancerNetwork, 0)
	q := manager.Query().Equals("loadbalancer_id", lbId)
	err := db.FetchModelObjects(manager, q, &lbnets)
	if err != nil {
		log.Errorf("getAllLoadbalancerNetworks fail %s", err)
		return nil, err
	}
	return lbnets, nil
}

func (lb *SLoadbalancer) detachAllNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	lbnets, err := LoadbalancernetworkManager.getAllLoadbalancerNetworks(lb.Id)
	if err != nil {
		return err
	}
	for i := range lbnets {
		err = lbnets[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *SLoadbalancer) purgeBackendGroups(ctx context.Context, userCred mcclient.TokenCredential) error {
	backendGroups, err := lb.GetLoadbalancerBackendgroups()
	if err != nil {
		return err
	}
	for i := range backendGroups {
		err = backendGroups[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *SLoadbalancer) purgeListeners(ctx context.Context, userCred mcclient.TokenCredential) error {
	listeners, err := lb.GetLoadbalancerListeners()
	if err != nil {
		return err
	}
	for i := range listeners {
		err = listeners[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *SLoadbalancer) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lb)
	defer lockman.ReleaseObject(ctx, lb)

	lb.DeletePreventionOff(lb, userCred)

	var err error

	if lb.BackendGroupId != "" {
		_, err = db.UpdateWithLock(ctx, lb, func() error {
			//避免 purge backendgroups 时循环依赖
			lb.BackendGroupId = ""
			return nil
		})

		if err != nil {
			return fmt.Errorf("loadbalancer %s(%s): clear up backend group error: %v", lb.Name, lb.Id, err)
		}
	}

	err = lb.detachAllNetworks(ctx, userCred)
	if err != nil {
		return err
	}

	err = lb.DeleteEip(ctx, userCred, false)
	if err != nil {
		return err
	}

	err = lb.purgeBackendGroups(ctx, userCred)
	if err != nil {
		return err
	}

	err = lb.purgeListeners(ctx, userCred)
	if err != nil {
		return err
	}

	err = lb.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	lb.LBPendingDelete(ctx, userCred)
	return nil
}

func (lbl *SLoadbalancerListener) purgeListenerRules(ctx context.Context, userCred mcclient.TokenCredential) error {
	listenerRules, err := lbl.GetLoadbalancerListenerRules()
	if err != nil {
		return err
	}

	for i := range listenerRules {
		err = listenerRules[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbl *SLoadbalancerListener) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbl)
	defer lockman.ReleaseObject(ctx, lbl)

	err := lbl.purgeListenerRules(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbl.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	lbl.LBPendingDelete(ctx, userCred)
	return nil
}

func (lblr *SLoadbalancerListenerRule) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lblr)
	defer lockman.ReleaseObject(ctx, lblr)

	err := lblr.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return lblr.DoPendingDelete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) purgeBackends(ctx context.Context, userCred mcclient.TokenCredential) error {
	backends, err := lbbg.GetBackends()
	if err != nil {
		return err
	}
	for i := range backends {
		err = backends[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) purgeListeners(ctx context.Context, userCred mcclient.TokenCredential) error {
	listeners, err := lbbg.GetLoadbalancerListeners()
	if err != nil {
		return err
	}
	for i := range listeners {
		err = listeners[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) purgeListenerrules(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules, err := lbbg.GetLoadbalancerListenerRules()
	if err != nil {
		return err
	}
	for i := range rules {
		err = rules[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.purgeBackends(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbbg.purgeListeners(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbbg.purgeListenerrules(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbbg.purgeCachedlbbg(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbbg.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	lbbg.LBPendingDelete(ctx, userCred)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) purgeCachedlbbg(ctx context.Context, userCred mcclient.TokenCredential) error {
	switch lbbg.GetProviderName() {
	case api.CLOUD_PROVIDER_AWS:
		return lbbg.purgeAwsCachedlbbg(ctx, userCred)
	case api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS:
		return lbbg.purgeHuaweiCachedlbbg(ctx, userCred)
	}

	return nil
}

func (lbbg *SLoadbalancerBackendGroup) purgeAwsCachedlbbg(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbbg.GetAwsCachedlbbg()
	if err != nil {
		return err
	}

	for i := range caches {
		if err := caches[i].purge(ctx, userCred); err != nil {
			return err
		}
	}

	return nil
}

func (lbbg *SAwsCachedLbbg) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return lbbg.SVirtualResourceBase.Delete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) purgeHuaweiCachedlbbg(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbbg.GetHuaweiCachedlbbg()
	if err != nil {
		return err
	}

	for i := range caches {
		if err := caches[i].purge(ctx, userCred); err != nil {
			return err
		}
	}

	return nil
}

func (lbbg *SHuaweiCachedLbbg) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return lbbg.SVirtualResourceBase.Delete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.purgeCachedlbb(ctx, userCred)
	if err != nil {
		return err
	}

	err = lbb.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return lbb.DoPendingDelete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) purgeCachedlbb(ctx context.Context, userCred mcclient.TokenCredential) error {
	switch lbb.GetProviderName() {
	case api.CLOUD_PROVIDER_AWS:
		return lbb.purgeAwsCachedlbb(ctx, userCred)
	case api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO:
		return lbb.purgeHuaweiCachedlbb(ctx, userCred)
	}

	return nil
}

func (lbb *SLoadbalancerBackend) purgeAwsCachedlbb(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbb.GetAwsCachedlbb()
	if err != nil {
		return err
	}

	for i := range caches {
		if err := caches[i].purge(ctx, userCred); err != nil {
			return err
		}
	}

	return nil
}

func (lbb *SAwsCachedLb) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return lbb.SVirtualResourceBase.Delete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) purgeHuaweiCachedlbb(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbb.GetHuaweiCachedlbb()
	if err != nil {
		return err
	}

	for i := range caches {
		if err := caches[i].purge(ctx, userCred); err != nil {
			return err
		}
	}

	return nil
}

func (lbb *SHuaweiCachedLb) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return lbb.SVirtualResourceBase.Delete(ctx, userCred)
}

func (manager *SSnapshotManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	snapshots := make([]SSnapshot, 0)
	err := fetchByManagerId(manager, providerId, &snapshots)
	if err != nil {
		return err
	}
	for i := range snapshots {
		err := snapshots[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (snapshot *SSnapshot) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, snapshot)
	defer lockman.ReleaseObject(ctx, snapshot)

	err := snapshot.ValidatePurgeCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "ValidatePurgeCondition for snapshot %s(%s)", snapshot.Name, snapshot.Id)
	}
	return snapshot.RealDelete(ctx, userCred)
}

func (manager *SSnapshotPolicyCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential,
	providerId string) error {
	// delete snapshot policy cache belong to manager
	spCaches := make([]SSnapshotPolicyCache, 0)
	err := fetchByManagerId(SnapshotPolicyCacheManager, providerId, &spCaches)
	if err != nil {
		return err
	}
	for i := range spCaches {
		err := spCaches[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (spc *SSnapshotPolicyCache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, spc)
	defer lockman.ReleaseObject(ctx, spc)
	err := spc.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return spc.RealDetele(ctx, userCred)
}

func (manager *SSnapshotPolicyManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	// delete snapshot policy cache belong to manager
	return SnapshotPolicyCacheManager.purgeAll(ctx, userCred, providerId)
}

func (manager *SStoragecacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	scs := make([]SStoragecache, 0)
	err := fetchByManagerId(manager, providerId, &scs)
	if err != nil {
		return err
	}
	for i := range scs {
		err := scs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *SStoragecache) purgeAllCachedimages(ctx context.Context, userCred mcclient.TokenCredential) error {
	cachedimages, err := sc.getCachedImages()
	if err != nil {
		return errors.Wrapf(err, "getCachedImages for storagecache %s(%s)", sc.Name, sc.Id)
	}
	for i := range cachedimages {
		err := cachedimages[i].syncRemoveCloudImage(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *SStoragecache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, sc)
	defer lockman.ReleaseObject(ctx, sc)

	err := sc.purgeAllCachedimages(ctx, userCred)
	if err != nil {
		return err
	}

	err = sc.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return sc.Delete(ctx, userCred)
}

func (manager *SStorageManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	storages := make([]SStorage, 0)
	err := fetchByManagerId(manager, providerId, &storages)
	if err != nil {
		return err
	}
	for i := range storages {
		err := storages[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (storage *SStorage) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	err := storage.purgeDisks(ctx, userCred)
	if err != nil {
		return err
	}

	err = storage.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return storage.Delete(ctx, userCred)
}

func (manager *SVpcManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	vpcs := make([]SVpc, 0)
	err := fetchByManagerId(manager, providerId, &vpcs)
	if err != nil {
		return err
	}
	for i := range vpcs {
		err := vpcs[i].Purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SGlobalVpcManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	gvpcs := make([]SGlobalVpc, 0)
	err := fetchByManagerId(manager, providerId, &gvpcs)
	if err != nil {
		return err
	}
	for i := range gvpcs {
		err := gvpcs[i].RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeGuestnetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := GuestnetworkManager.Query().Equals("network_id", net.Id)
	gns := make([]SGuestnetwork, 0)
	err := db.FetchModelObjects(GuestnetworkManager, q, &gns)
	if err != nil {
		return err
	}
	for i := range gns {
		err = gns[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeHostnetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := HostnetworkManager.Query().Equals("network_id", net.Id)
	hns := make([]SHostnetwork, 0)
	err := db.FetchModelObjects(HostnetworkManager, q, &hns)
	if err != nil {
		return err
	}
	for i := range hns {
		err = hns[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeGroupnetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := GroupnetworkManager.Query().Equals("network_id", net.Id)
	grns := make([]SGroupnetwork, 0)
	err := db.FetchModelObjects(GroupnetworkManager, q, &grns)
	if err != nil {
		return err
	}
	for i := range grns {
		err = grns[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeLoadbalancernetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := LoadbalancernetworkManager.Query().Equals("network_id", net.Id)
	lbns := make([]SLoadbalancerNetwork, 0)
	err := db.FetchModelObjects(LoadbalancernetworkManager, q, &lbns)
	if err != nil {
		return err
	}
	for i := range lbns {
		err = lbns[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeEipnetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := ElasticipManager.Query().Equals("network_id", net.Id)
	eips := make([]SElasticip, 0)
	err := db.FetchModelObjects(ElasticipManager, q, &eips)
	if err != nil {
		return err
	}
	for i := range eips {
		err = eips[i].RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purgeReservedIps(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := ReservedipManager.Query().Equals("network_id", net.Id)
	rips := make([]SReservedip, 0)
	err := db.FetchModelObjects(ReservedipManager, q, &rips)
	if err != nil {
		return err
	}
	for i := range rips {
		err = rips[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nic *SNetworkInterface) purgeNetworkAddres(ctx context.Context, userCred mcclient.TokenCredential) error {
	networks, err := nic.GetNetworks()
	if err != nil {
		return err
	}

	for i := range networks {
		err := networks[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nic *SNetworkInterface) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, nic)
	defer lockman.ReleaseObject(ctx, nic)

	err := nic.purgeNetworkAddres(ctx, userCred)
	if err != nil {
		return err
	}

	err = nic.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return nic.Delete(ctx, userCred)
}

func (net *SNetwork) purgeDBInstanceNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	dbNets, err := net.GetDBInstanceNetworks()
	if err != nil {
		return errors.Wrapf(err, "GetDBInstanceNetworks")
	}
	for i := range dbNets {
		err = dbNets[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete %d", dbNets[i].RowId)
		}
	}
	return nil
}

func (net *SNetwork) purgeNetworkInterfaces(ctx context.Context, userCred mcclient.TokenCredential) error {
	networkinterfaceIds := NetworkinterfacenetworkManager.Query("networkinterface_id").Equals("network_id", net.Id).Distinct().SubQuery()
	q := NetworkInterfaceManager.Query().In("id", networkinterfaceIds)
	interfaces := make([]SNetworkInterface, 0)
	err := db.FetchModelObjects(NetworkInterfaceManager, q, &interfaces)
	if err != nil {
		return err
	}
	for i := range interfaces {
		err = interfaces[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (net *SNetwork) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)

	err := net.purgeGuestnetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeGuestnetworks")
	}
	err = net.purgeHostnetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeHostnetworks")
	}
	err = net.purgeGroupnetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeGroupnetworks")
	}
	err = net.purgeLoadbalancernetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeLoadbalancernetworks")
	}
	err = net.purgeEipnetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeEipnetworks")
	}
	err = net.purgeReservedIps(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeReservedIps")
	}

	err = net.purgeNetworkInterfaces(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeNetworkInterfaces")
	}

	err = net.purgeDBInstanceNetworks(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeDBInstanceNetworks")
	}

	err = net.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	return net.RealDelete(ctx, userCred)
}

func (wire *SWire) purgeNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	nets, err := wire.getNetworks(nil, rbacutils.ScopeNone)
	if err != nil {
		return err
	}
	for i := range nets {
		err := nets[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wire *SWire) purgeHostwires(ctx context.Context, userCred mcclient.TokenCredential) error {
	hws, _ := wire.GetHostwires()
	for j := range hws {
		err := hws[j].Detach(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wire *SWire) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, wire)
	defer lockman.ReleaseObject(ctx, wire)

	err := wire.purgeNetworks(ctx, userCred)
	if err != nil {
		return err
	}
	err = wire.purgeHostwires(ctx, userCred)
	if err != nil {
		return err
	}
	err = wire.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return wire.Delete(ctx, userCred)
}

func (vpc *SVpc) purgeWires(ctx context.Context, userCred mcclient.TokenCredential) error {
	wires, err := vpc.GetWires()
	if err != nil {
		return errors.Wrapf(err, "GetWres for vpc %s", vpc.Id)
	}
	for i := range wires {
		err := wires[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SVpc) purgeIPv6Gateways(ctx context.Context, userCred mcclient.TokenCredential) error {
	gws, err := self.GetIPv6Gateways()
	if err != nil {
		return errors.Wrapf(err, "GetIPv6Gateways for vpc %s", self.Id)
	}
	for i := range gws {
		err := gws[i].RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vpc *SVpc) purgeVpcPeeringConnections(ctx context.Context, userCred mcclient.TokenCredential) error {
	vpcPCs, err := vpc.GetVpcPeeringConnections()
	if err != nil {
		return err
	}
	for i := range vpcPCs {
		err := vpcPCs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vpc *SVpc) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, vpc)
	defer lockman.ReleaseObject(ctx, vpc)

	err := vpc.purgeVpcPeeringConnections(ctx, userCred)
	if err != nil {
		return err
	}

	err = vpc.purgeIPv6Gateways(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeIPv6Gateways")
	}

	err = vpc.purgeWires(ctx, userCred)
	if err != nil {
		return err
	}
	err = vpc.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return vpc.RealDelete(ctx, userCred)
}

func (dn *SNatDEntry) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := dn.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return dn.RealDelete(ctx, userCred)
}

func (sn *SNatSEntry) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := sn.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return sn.RealDelete(ctx, userCred)
}

func (manager *SCloudproviderregionManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	cprs, err := CloudproviderRegionManager.fetchRecordsByCloudproviderId(providerId)
	if err != nil {
		return err
	}
	for i := range cprs {
		err = cprs[i].Detach(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (zone *SZone) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, zone)
	defer lockman.ReleaseObject(ctx, zone)

	err := zone.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return zone.Delete(ctx, userCred)
}

func (self *SCloudregion) purgeSkus(ctx context.Context, userCred mcclient.TokenCredential) error {
	skus, err := self.GetServerSkus()
	if err != nil {
		return errors.Wrapf(err, "GetServerSkus")
	}
	for i := range skus {
		err = skus[i].RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeMiscResources(ctx context.Context, userCred mcclient.TokenCredential) error {
	misc, err := self.GetMiscResources()
	if err != nil {
		return errors.Wrapf(err, "GetServerSkus")
	}
	for i := range misc {
		err = misc[i].RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (region *SCloudregion) purgeZones(ctx context.Context, userCred mcclient.TokenCredential) error {
	zones, err := region.GetZones()
	if err != nil {
		return err
	}
	for i := range zones {
		err = zones[i].Purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (region *SCloudregion) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, region)
	defer lockman.ReleaseObject(ctx, region)

	err := region.purgeSkus(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeSkus")
	}

	err = region.purgeZones(ctx, userCred)
	if err != nil {
		return err
	}

	err = region.purgeMiscResources(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeMiscResources")
	}

	err = region.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return region.Delete(ctx, userCred)
}

func (manager *SCloudregionManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	regions, err := manager.getCloudregionsByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range regions {
		c, err := CloudproviderRegionManager.Query().NotEquals("cloudprovider_id", providerId).Equals("cloudregion_id", regions[i].GetId()).CountWithError()
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			return err
		}
		// 仍然有其他provider在使用该region
		if c > 0 {
			log.Infof("cloud region is using by other providers.skipped purge region %s with id %s", regions[i].GetName(), regions[i].GetId())
			return nil
		}

		err = regions[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (table *SNatSEntry) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, table)
	defer lockman.ReleaseObject(ctx, table)

	err := table.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return table.RealDelete(ctx, userCred)
}

func (nat *SNatGateway) purgeSTables(ctx context.Context, userCred mcclient.TokenCredential) error {
	tables, err := nat.GetSTable()
	if err != nil {
		return err
	}

	for i := range tables {
		err = tables[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (table *SNatDEntry) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, table)
	defer lockman.ReleaseObject(ctx, table)

	err := table.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return table.RealDelete(ctx, userCred)
}

func (nat *SNatGateway) purgeDTables(ctx context.Context, userCred mcclient.TokenCredential) error {
	tables, err := nat.GetDTable()
	if err != nil {
		return err
	}

	for i := range tables {
		err = tables[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nat *SNatGateway) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, nat)
	defer lockman.ReleaseObject(ctx, nat)

	err := nat.purgeDTables(ctx, userCred)
	if err != nil {
		return err
	}

	err = nat.purgeSTables(ctx, userCred)
	if err != nil {
		return err
	}

	nat.DeletePreventionOff(nat, userCred)

	err = nat.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return nat.RealDelete(ctx, userCred)
}

func (manager *SNatGatewayManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	nats, err := manager.getNatgatewaysByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range nats {
		err = nats[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SNetworkInterfaceManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	nics, err := manager.getNetworkInterfacesByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range nics {
		err = nics[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bucket *SBucket) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, bucket)
	defer lockman.ReleaseObject(ctx, bucket)

	err := bucket.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return bucket.RealDelete(ctx, userCred)
}

func (bucketManager *SBucketManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	buckets := make([]SBucket, 0)
	err := fetchByManagerId(bucketManager, providerId, &buckets)
	if err != nil {
		return err
	}
	for i := range buckets {
		err := buckets[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (account *SDBInstanceAccount) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, account)
	defer lockman.ReleaseObject(ctx, account)

	privileges, err := account.GetDBInstancePrivileges()
	if err != nil {
		return errors.Wrap(err, "account.GetDBInstancePrivileges")
	}

	for _, privilege := range privileges {
		err = privilege.Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "privilege.Delete() %s", privilege.Id)
		}
	}

	err = account.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return account.RealDelete(ctx, userCred)
}

func (instance *SDBInstance) purgeAccounts(ctx context.Context, userCred mcclient.TokenCredential) error {
	accounts, err := instance.GetDBInstanceAccounts()
	if err != nil {
		return err
	}

	for i := range accounts {
		err = accounts[i].Purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (database *SDBInstanceDatabase) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, database)
	defer lockman.ReleaseObject(ctx, database)

	privileges, err := database.GetDBInstancePrivileges()
	if err != nil {
		return errors.Wrap(err, "database.GetDBInstancePrivileges")
	}

	for _, privilege := range privileges {
		err = privilege.Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "privilege.Delete() %s", privilege.Id)
		}
	}

	err = database.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return database.RealDelete(ctx, userCred)
}

func (instance *SDBInstance) purgeDatabases(ctx context.Context, userCred mcclient.TokenCredential) error {
	databases, err := instance.GetDBDatabases()
	if err != nil {
		return err
	}

	for i := range databases {
		err = databases[i].Purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (parameter *SDBInstanceParameter) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, parameter)
	defer lockman.ReleaseObject(ctx, parameter)

	err := parameter.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return parameter.Delete(ctx, userCred)
}

func (instance *SDBInstance) purgeParameters(ctx context.Context, userCred mcclient.TokenCredential) error {
	parameters, err := instance.GetDBParameters()
	if err != nil {
		return err
	}

	for i := range parameters {
		err = parameters[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (network *SDBInstanceNetwork) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)

	err := network.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	return network.Delete(ctx, userCred)
}

func (instance *SDBInstance) purgeNetwork(ctx context.Context, userCred mcclient.TokenCredential) error {
	networks, err := instance.GetDBNetworks()
	if err != nil {
		return errors.Wrapf(err, "GetDBNetworks")
	}
	for i := range networks {
		err = networks[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "networks.purge %d", networks[i].RowId)
		}
	}
	return nil
}

func (self *SDBInstanceSecgroup) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.Detach(ctx, userCred)
}

func (instance *SDBInstance) purgeSecgroups(ctx context.Context, userCred mcclient.TokenCredential) error {
	secgroups, err := instance.GetDBInstanceSecgroups()
	if err != nil {
		return errors.Wrapf(err, "GetDBInstanceSecgroups")
	}
	for i := range secgroups {
		err = secgroups[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "secgroups.purge %d", secgroups[i].RowId)
		}
	}
	return nil
}

func (instance *SDBInstance) PurgeBackups(ctx context.Context, userCred mcclient.TokenCredential, mode string) error {
	backups, err := instance.GetDBInstanceBackupByMode(mode)
	if err != nil {
		return errors.Wrap(err, "instance.GetDBInstanceBackups")
	}
	for _, backup := range backups {
		err = backup.purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "backup.purge %s(%s)", backup.Name, backup.Id)
		}
	}
	return nil
}

func (instance *SDBInstance) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	instance.DeletePreventionOff(instance, userCred)

	err := instance.purgeAccounts(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeAccounts")
	}

	err = instance.purgeDatabases(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeDatabases")
	}

	err = instance.purgeParameters(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeParameters")
	}

	err = instance.purgeNetwork(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeNetwork")
	}

	err = instance.purgeSecgroups(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "purgeSecgroups")
	}

	err = instance.PurgeBackups(ctx, userCred, api.BACKUP_MODE_AUTOMATED)
	if err != nil {
		return errors.Wrap(err, "instance.purgeBackups")
	}

	err = instance.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}

	return instance.RealDelete(ctx, userCred)
}

func (manager *SDBInstanceManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	instances, err := manager.getDBInstancesByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range instances {
		err = instances[i].Purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (backup *SDBInstanceBackup) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, backup)
	defer lockman.ReleaseObject(ctx, backup)

	err := backup.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	return backup.RealDelete(ctx, userCred)
}

func (manager *SDBInstanceBackupManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	backups, err := manager.getDBInstanceBackupsByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range backups {
		err = backups[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SElasticcache) purgeAccounts(ctx context.Context, userCred mcclient.TokenCredential) error {
	accounts, err := instance.GetElasticcacheAccounts()
	if err != nil {
		return err
	}

	for i := range accounts {
		err = accounts[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SElasticcacheAccount) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	err := instance.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return instance.Delete(ctx, userCred)
}

func (instance *SElasticcache) purgeAcls(ctx context.Context, userCred mcclient.TokenCredential) error {
	acls, err := instance.GetElasticcacheAcls()
	if err != nil {
		return err
	}

	for i := range acls {
		err = acls[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SElasticcacheAcl) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	err := instance.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return instance.Delete(ctx, userCred)
}

func (instance *SElasticcache) purgeBackups(ctx context.Context, userCred mcclient.TokenCredential) error {
	backups, err := instance.GetElasticcacheBackups()
	if err != nil {
		return err
	}

	for i := range backups {
		err = backups[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SElasticcacheBackup) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	err := instance.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return instance.Delete(ctx, userCred)
}

func (instance *SElasticcache) purgeParameters(ctx context.Context, userCred mcclient.TokenCredential) error {
	parameters, err := instance.GetElasticcacheParameters()
	if err != nil {
		return err
	}

	for i := range parameters {
		err = parameters[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SElasticcacheParameter) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	err := instance.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return instance.Delete(ctx, userCred)
}

func (instance *SElasticcache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	instance.DeletePreventionOff(instance, userCred)

	err := instance.purgeAccounts(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeAcls(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeBackups(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeParameters(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}

	return instance.SVirtualResourceBase.Delete(ctx, userCred)
}

func (manager *SElasticcacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	instances, err := manager.getElasticcachesByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range instances {
		err = instances[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SSecurityGroupCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	caches := []SSecurityGroupCache{}
	err := fetchByManagerId(manager, providerId, &caches)
	if err != nil {
		return err
	}
	for i := range caches {
		err := caches[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (quota *SCloudproviderQuota) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, quota)
	defer lockman.ReleaseObject(ctx, quota)

	err := quota.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return quota.Delete(ctx, userCred)
}

func (manager *SCloudproviderQuotaManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	quotas := []SCloudproviderQuota{}
	err := fetchByManagerId(manager, providerId, &quotas)
	if err != nil {
		return err
	}
	for i := range quotas {
		err := quotas[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (assignment *SPolicyAssignment) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, assignment)
	defer lockman.ReleaseObject(ctx, assignment)

	err := assignment.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "assignment.ValidateDeleteCondition(%s(%s))", assignment.Name, assignment.Id)
	}

	return assignment.Delete(ctx, userCred)
}

func (definition *SPolicyDefinition) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, definition)
	defer lockman.ReleaseObject(ctx, definition)

	assignments, err := definition.GetPolicyAssignments()
	if err != nil {
		return errors.Wrap(err, "definition.GetPolicyAssignments")
	}

	for i := range assignments {
		err = assignments[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}

	err = definition.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}

	return definition.Delete(ctx, userCred)
}

func (manager *SPolicyDefinitionManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	definitions := []SPolicyDefinition{}
	err := fetchByManagerId(manager, providerId, &definitions)
	if err != nil {
		return err
	}
	for i := range definitions {
		err := definitions[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SAccessGroupCache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.RealDelete(ctx, userCred)
}

func (manager *SAccessGroupCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	caches := []SAccessGroupCache{}
	err := fetchByManagerId(manager, providerId, &caches)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}

	for i := range caches {
		err := caches[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache purge")
		}
	}

	return nil
}

func (self *SFileSystem) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.RealDelete(ctx, userCred)
}

func (manager *SFileSystemManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	files := []SFileSystem{}
	err := fetchByManagerId(manager, providerId, &files)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}

	for i := range files {
		err := files[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache purge")
		}
	}

	return nil
}

func (vpcPC *SVpcPeeringConnection) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, vpcPC)
	defer lockman.ReleaseObject(ctx, vpcPC)
	return vpcPC.RealDelete(ctx, userCred)
}

func (manager *SInterVpcNetworkManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	networks := []SInterVpcNetwork{}
	err := fetchByManagerId(manager, providerId, &networks)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range networks {
		err := networks[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "inter vpc network delete")
		}
	}
	return nil
}

func (manager *SWafRuleGroupCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	caches := []SWafRuleGroupCache{}
	err := fetchByManagerId(manager, providerId, &caches)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range caches {
		err := caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache delete")
		}
	}
	return nil
}

func (manager *SWafIPSetCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	caches := []SWafIPSetCache{}
	err := fetchByManagerId(manager, providerId, &caches)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range caches {
		err := caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache delete")
		}
	}
	return nil
}

func (manager *SWafRegexSetCacheManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	caches := []SWafRegexSetCache{}
	err := fetchByManagerId(manager, providerId, &caches)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range caches {
		err := caches[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache delete")
		}
	}
	return nil
}

func (manager *SWafInstanceManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	ins := []SWafInstance{}
	err := fetchByManagerId(manager, providerId, &ins)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range ins {
		err := ins[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache delete")
		}
	}
	return nil
}

func (manager *SMongoDBManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	dbs := []SMongoDB{}
	err := fetchByManagerId(manager, providerId, &dbs)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range dbs {
		err := dbs[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cache delete")
		}
	}
	return nil
}

func (manager *SElasticSearchManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	ess := []SElasticSearch{}
	err := fetchByManagerId(manager, providerId, &ess)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range ess {
		lockman.LockObject(ctx, &ess[i])
		defer lockman.ReleaseObject(ctx, &ess[i])

		err := ess[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "elastic search delete")
		}
	}
	return nil
}

func (manager *SKafkaManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	kafkas := []SKafka{}
	err := fetchByManagerId(manager, providerId, &kafkas)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range kafkas {
		lockman.LockObject(ctx, &kafkas[i])
		defer lockman.ReleaseObject(ctx, &kafkas[i])

		err := kafkas[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "kafka delete")
		}
	}
	return nil
}

func (manager *SCDNDomainManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	domains := []SCDNDomain{}
	err := fetchByManagerId(manager, providerId, &domains)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range domains {
		lockman.LockObject(ctx, &domains[i])
		defer lockman.ReleaseObject(ctx, &domains[i])

		err := domains[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "cdn domain delete")
		}
	}
	return nil
}

func (manager *SKubeClusterManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	clusters := []SKubeCluster{}
	err := fetchByManagerId(manager, providerId, &clusters)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range clusters {
		lockman.LockObject(ctx, &clusters[i])
		defer lockman.ReleaseObject(ctx, &clusters[i])

		err := clusters[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "kube cluster delete")
		}
	}
	return nil
}

func (manager *STablestoreManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	tablestores := make([]STablestore, 0)
	err := fetchByManagerId(manager, providerId, &tablestores)
	if err != nil {
		return err
	}
	for i := range tablestores {
		lockman.LockObject(ctx, &tablestores[i])
		defer lockman.ReleaseObject(ctx, &tablestores[i])

		err := tablestores[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "real delete %s", tablestores[i].Id)
		}
	}
	return nil
}

func (manager *SModelartsPoolManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	ess := []SModelartsPool{}
	err := fetchByManagerId(manager, providerId, &ess)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range ess {
		lockman.LockObject(ctx, &ess[i])
		defer lockman.ReleaseObject(ctx, &ess[i])

		err := ess[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "modelarts pool delete")
		}
	}
	return nil
}

func (manager *SModelartsPoolSkuManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	poolSku := []SModelartsPoolSku{}
	err := fetchByManagerId(manager, providerId, &poolSku)
	if err != nil {
		return errors.Wrapf(err, "fetchByManagerId")
	}
	for i := range poolSku {
		lockman.LockObject(ctx, &poolSku[i])
		defer lockman.ReleaseObject(ctx, &poolSku[i])

		err := poolSku[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "modelarts pool delete")
		}
	}
	return nil
}
