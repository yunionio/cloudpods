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

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

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

	err := eip.ValidateDeleteCondition(ctx)
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

	_, err := host.PerformDisable(ctx, userCred, nil, nil)
	if err != nil {
		return err
	}

	guests := host.GetGuests()
	for i := range guests {
		err := guests[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}

	// clean all disks on locally attached storages
	storages := host._getAttachedStorages(tristate.None, tristate.None, api.STORAGE_LOCAL)
	for i := range storages {
		err := storages[i].purgeDisks(ctx, userCred)
		if err != nil {
			return err
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
	guest.EjectIso(userCred)
	guest.DeleteEip(ctx, userCred)
	guest.DeleteAllDisksInDB(ctx, userCred)

	err := guest.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return guest.RealDelete(ctx, userCred)
}

func (storage *SStorage) purgeDisks(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	disks := storage.GetDisks()
	for i := range disks {
		err := disks[i].purge(ctx, userCred)
		if err != nil {
			return err
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

func (manager *SLoadbalancerCertificateManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbcs := make([]SLoadbalancerCertificate, 0)
	err := fetchByManagerId(manager, providerId, &lbcs)
	if err != nil {
		return err
	}
	for i := range lbcs {
		err := lbcs[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbcert *SLoadbalancerCertificate) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbcert)
	defer lockman.ReleaseObject(ctx, lbcert)

	err := lbcert.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return lbcert.DoPendingDelete(ctx, userCred)
}

func (manager *SLoadbalancerAclManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	lbacls := make([]SLoadbalancerAcl, 0)
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

func (lbacl *SLoadbalancerAcl) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbacl)
	defer lockman.ReleaseObject(ctx, lbacl)

	err := lbacl.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return lbacl.DoPendingDelete(ctx, userCred)
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

	_, err := db.UpdateWithLock(ctx, lb, func() error {
		//避免 purge backendgroups 时循环依赖
		lb.BackendGroupId = ""
		return nil
	})

	if err != nil {
		return fmt.Errorf("loadbalancer %s(%s): clear up backend group error: %v", lb.Name, lb.Id, err)
	}

	err = lb.detachAllNetworks(ctx, userCred)
	if err != nil {
		return err
	}

	err = lb.DeleteEip(ctx, userCred)
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

	err = lb.ValidateDeleteCondition(ctx)
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

	err = lbl.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	lbl.LBPendingDelete(ctx, userCred)
	return nil
}

func (lblr *SLoadbalancerListenerRule) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lblr)
	defer lockman.ReleaseObject(ctx, lblr)

	err := lblr.ValidateDeleteCondition(ctx)
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

	err = lbbg.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	lbbg.LBPendingDelete(ctx, userCred)
	return nil
}

func (lbb *SLoadbalancerBackend) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return lbb.DoPendingDelete(ctx, userCred)
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

	err := snapshot.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return snapshot.RealDelete(ctx, userCred)
}

func (manager *SSnapshotPolicyManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	sps := make([]SSnapshotPolicy, 0)
	err := fetchByManagerId(manager, providerId, &sps)
	if err != nil {
		return err
	}
	for i := range sps {
		err := sps[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sp *SSnapshotPolicy) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, sp)
	defer lockman.ReleaseObject(ctx, sp)

	err := sp.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return sp.RealDelete(ctx, userCred)
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
	cachedimages := sc.getCachedImages()
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

	err = sc.ValidateDeleteCondition(ctx)
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

	err = storage.ValidateDeleteCondition(ctx)
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

	err = nic.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return nic.Delete(ctx, userCred)
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
		return err
	}
	err = net.purgeHostnetworks(ctx, userCred)
	if err != nil {
		return err
	}
	err = net.purgeGroupnetworks(ctx, userCred)
	if err != nil {
		return err
	}
	err = net.purgeLoadbalancernetworks(ctx, userCred)
	if err != nil {
		return err
	}
	err = net.purgeEipnetworks(ctx, userCred)
	if err != nil {
		return err
	}
	err = net.purgeReservedIps(ctx, userCred)
	if err != nil {
		return err
	}

	err = net.purgeNetworkInterfaces(ctx, userCred)
	if err != nil {
		return err
	}

	err = net.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return net.RealDelete(ctx, userCred)
}

func (wire *SWire) purgeNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	nets, err := wire.getNetworks()
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
	err = wire.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return wire.Delete(ctx, userCred)
}

func (vpc *SVpc) purgeWires(ctx context.Context, userCred mcclient.TokenCredential) error {
	wires := vpc.GetWires()
	for i := range wires {
		err := wires[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vpc *SVpc) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, vpc)
	defer lockman.ReleaseObject(ctx, vpc)

	err := vpc.purgeWires(ctx, userCred)
	if err != nil {
		return err
	}
	err = vpc.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return vpc.RealDelete(ctx, userCred)
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

func (manager *SExternalProjectManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	projs, err := manager.getProjectsByProviderId(providerId)
	if err != nil {
		return err
	}
	for i := range projs {
		err = projs[i].Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (zone *SZone) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, zone)
	defer lockman.ReleaseObject(ctx, zone)

	err := zone.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return zone.Delete(ctx, userCred)
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

func (sku *SServerSku) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, sku)
	defer lockman.ReleaseObject(ctx, sku)

	err := sku.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return sku.Delete(ctx, userCred)
}

func (region *SCloudregion) purgeSkus(ctx context.Context, userCred mcclient.TokenCredential) error {
	skus, err := region.GetSkus()
	if err != nil {
		return err
	}
	for i := range skus {
		err = skus[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (region *SCloudregion) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, region)
	defer lockman.ReleaseObject(ctx, region)

	err := region.purgeZones(ctx, userCred)
	if err != nil {
		return err
	}

	err = region.purgeSkus(ctx, userCred)
	if err != nil {
		return err
	}

	err = region.ValidateDeleteCondition(ctx)
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

	err := table.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return table.Delete(ctx, userCred)
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

	err := table.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return table.Delete(ctx, userCred)
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

	err = nat.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return nat.Delete(ctx, userCred)
}

func (manager *SNatGetewayManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
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

	err := bucket.ValidateDeleteCondition(ctx)
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

func (privilege *SDBInstancePrivilege) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, privilege)
	defer lockman.ReleaseObject(ctx, privilege)

	err := privilege.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return privilege.Delete(ctx, userCred)
}

func (account *SDBInstanceAccount) purgePrivileges(ctx context.Context, userCred mcclient.TokenCredential) error {
	privileges, err := account.GetDBInstancePrivileges()
	if err != nil {
		return err
	}

	for i := range privileges {
		err = privileges[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (account *SDBInstanceAccount) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, account)
	defer lockman.ReleaseObject(ctx, account)

	err := account.purgePrivileges(ctx, userCred)
	if err != nil {
		return err
	}

	err = account.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return account.Delete(ctx, userCred)
}

func (instance *SDBInstance) purgeAccounts(ctx context.Context, userCred mcclient.TokenCredential) error {
	accounts, err := instance.GetDBInstanceAccounts()
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

func (database *SDBInstanceDatabase) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, database)
	defer lockman.ReleaseObject(ctx, database)

	err := database.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return database.Delete(ctx, userCred)
}

func (instance *SDBInstance) purgeDatabases(ctx context.Context, userCred mcclient.TokenCredential) error {
	databases, err := instance.GetDBDatabases()
	if err != nil {
		return err
	}

	for i := range databases {
		err = databases[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (parameter *SDBInstanceParameter) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, parameter)
	defer lockman.ReleaseObject(ctx, parameter)

	err := parameter.ValidateDeleteCondition(ctx)
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

	err := network.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return network.Delete(ctx, userCred)
}

func (instance *SDBInstance) purgeNetwork(ctx context.Context, userCred mcclient.TokenCredential) error {
	network, _ := instance.GetDBNetwork()
	if network != nil {
		return network.purge(ctx, userCred)
	}
	return nil
}

func (instance *SDBInstance) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

	err := instance.purgeAccounts(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeDatabases(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeParameters(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.purgeNetwork(ctx, userCred)
	if err != nil {
		return err
	}

	err = instance.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return instance.Delete(ctx, userCred)
}

func (manager *SDBInstanceManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	instances, err := manager.getDBInstancesByProviderId(providerId)
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

func (backup *SDBInstanceBackup) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, backup)
	defer lockman.ReleaseObject(ctx, backup)

	err := backup.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return backup.Delete(ctx, userCred)
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

	err := instance.ValidateDeleteCondition(ctx)
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

	err := instance.ValidateDeleteCondition(ctx)
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

	err := instance.ValidateDeleteCondition(ctx)
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

	err := instance.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	return instance.Delete(ctx, userCred)
}

func (instance *SElasticcache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, instance)
	defer lockman.ReleaseObject(ctx, instance)

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

	err = instance.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	return instance.Delete(ctx, userCred)
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
