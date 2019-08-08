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
	err = net.purgeReservedIps(ctx, userCred)
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
