package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-plus/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func (self *SHost) GetResourceType() string {
	if len(self.ResourceType) > 0 {
		return self.ResourceType
	}
	return HostResourceTypeDefault
}

func (self *SGuest) AllowPerformPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "prepaid-recycle")
}

func (self *SGuest) CanPerformPrepaidRecycle() error {
	if self.BillingType != BILLING_TYPE_PREPAID {
		return fmt.Errorf("recycle prepaid server only")
	}
	if self.ExpiredAt.Before(time.Now()) {
		return fmt.Errorf("prepaid expired")
	}
	host := self.GetHost()
	if host == nil {
		return fmt.Errorf("no host")
	}
	if !host.IsManaged() {
		return fmt.Errorf("only managed prepaid server can be pooled")
	}
	return nil
}

func (self *SGuest) PerformPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.CanPerformPrepaidRecycle()
	if err != nil {
		return nil, httperrors.NewInvalidStatusError(err.Error())
	}

	err = self.doPrepaidRecycle(ctx, userCred)
	if err != nil {
		logclient.AddActionLog(self, logclient.ACT_RECYCLE_PREPAID, self.GetShortDesc(), userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}

	db.OpsLog.LogEvent(self, db.ACT_RECYCLE_PREPAID, self.GetShortDesc(), userCred)
	logclient.AddActionLog(self, logclient.ACT_RECYCLE_PREPAID, self.GetShortDesc(), userCred, true)

	return nil, nil
}

func (self *SGuest) doPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockClass(ctx, HostManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, HostManager, userCred.GetProjectId())

	return self.doPrepaidRecycleNoLock(ctx, userCred)
}

func (self *SGuest) doPrepaidRecycleNoLock(ctx context.Context, userCred mcclient.TokenCredential) error {
	oHost := self.GetHost()

	fakeHost := SHost{}
	fakeHost.SetModelManager(HostManager)

	fakeHost.Name = fmt.Sprintf("%s-host", self.Name)
	fakeHost.CpuCount = self.VcpuCount
	fakeHost.NodeCount = 1
	fakeHost.CpuCmtbound = 1.0

	fakeHost.MemCmtbound = 1.0
	fakeHost.MemReserved = 0
	fakeHost.MemSize = self.VmemSize

	guestdisks := self.GetDisks()

	storageInfo := make([]baremetal.BaremetalStorage, 0)
	totalSize := 0
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		storage := disk.GetStorage()

		totalSize += disk.DiskSize

		if len(fakeHost.StorageType) == 0 {
			fakeHost.StorageType = storage.StorageType
		}

		info := baremetal.BaremetalStorage{}
		info.Size = int64(disk.DiskSize)
		info.Index = int64(i)
		info.Slot = i
		info.Driver = baremetal.DISK_DRIVER_LINUX
		info.Rotate = (storage.MediumType != DISK_TYPE_SSD)

		storageInfo = append(storageInfo, info)
	}

	fakeHost.StorageDriver = baremetal.DISK_DRIVER_LINUX
	fakeHost.StorageSize = totalSize
	fakeHost.StorageInfo = jsonutils.Marshal(&storageInfo)

	fakeHost.ZoneId = self.getZone().GetId()
	fakeHost.IsBaremetal = false
	fakeHost.IsMaintenance = false
	fakeHost.ResourceType = HostResourceTypePrepaidRecycle

	guestnics := self.GetNetworks()
	if len(guestnics) == 0 {
		msg := "no network info on guest????"
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}
	fakeHost.AccessIp = guestnics[0].IpAddr
	fakeHost.AccessMac = guestnics[0].MacAddr

	fakeHost.BillingType = BILLING_TYPE_PREPAID
	fakeHost.BillingCycle = self.BillingCycle
	fakeHost.ExpiredAt = self.ExpiredAt

	fakeHost.Status = HOST_STATUS_RUNNING
	fakeHost.HostStatus = HOST_ONLINE
	fakeHost.Enabled = true
	fakeHost.HostType = oHost.HostType
	fakeHost.ExternalId = oHost.ExternalId
	fakeHost.RealExternalId = self.ExternalId
	fakeHost.ManagerId = oHost.ManagerId
	fakeHost.IsEmulated = true
	fakeHost.Description = "fake host for prepaid vm recycling"

	err := HostManager.TableSpec().Insert(&fakeHost)
	if err != nil {
		log.Errorf("fail to insert fake host %s", err)
		return err
	}

	for i := 0; i < len(guestnics); i += 1 {
		var nicType string
		if i == 0 {
			nicType = NIC_TYPE_ADMIN
		}
		err = fakeHost.addNetif(ctx, userCred,
			guestnics[i].MacAddr,
			guestnics[i].GetNetwork().WireId,
			"",
			1000,
			nicType,
			int8(i),
			tristate.True,
			1500,
			false,
			fmt.Sprintf("eth%d", i),
			fmt.Sprintf("br%d", i),
			false,
			false)
		if err != nil {
			log.Errorf("fail to addNetInterface %d: %s", i, err)
			fakeHost.RealDelete(ctx, userCred)
			return err
		}
	}

	storageSize := 0
	var externalId string
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		storage := disk.GetStorage()
		if disk.BillingType == BILLING_TYPE_PREPAID {
			storageSize += disk.DiskSize
			if len(externalId) == 0 {
				externalId = storage.ExternalId
			} else {
				if externalId != storage.ExternalId {
					msg := "inconsistent storage !!!!"
					log.Errorf(msg)
					return errors.New(msg)
				}
			}
		}
	}

	sysStorage := guestdisks[0].GetDisk().GetStorage()

	fakeStorage := SStorage{}
	fakeStorage.SetModelManager(StorageManager)

	fakeStorage.Name = fmt.Sprintf("%s-storage", self.Name)
	fakeStorage.Capacity = storageSize
	fakeStorage.StorageType = STORAGE_LOCAL
	fakeStorage.MediumType = sysStorage.MediumType
	fakeStorage.Cmtbound = 1.0
	fakeStorage.ZoneId = fakeHost.ZoneId
	fakeStorage.StoragecacheId = sysStorage.StoragecacheId
	fakeStorage.Enabled = true
	fakeStorage.Status = STORAGE_ONLINE
	fakeStorage.Description = "fake storage for prepaid vm recycling"
	fakeStorage.IsEmulated = true
	fakeStorage.ManagerId = sysStorage.ManagerId
	fakeStorage.ExternalId = externalId

	err = StorageManager.TableSpec().Insert(&fakeStorage)
	if err != nil {
		log.Errorf("fail to insert fake storage %s", err)
		return err
	}

	err = fakeHost.Attach2Storage(ctx, userCred, &fakeStorage, "")
	if err != nil {
		log.Errorf("fail to add fake storage: %s", err)
		fakeHost.RealDelete(ctx, userCred)
		return err
	}

	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		// clear billing information
		self.BillingType = BILLING_TYPE_POSTPAID
		self.BillingCycle = ""
		self.ExpiredAt = time.Time{}
		// switch to fakeHost
		self.HostId = fakeHost.Id
		return nil
	})

	if err != nil {
		log.Errorf("clear billing information fail: %s", err)
		fakeHost.RealDelete(ctx, userCred)
		return err
	}

	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()

		if disk.BillingType == BILLING_TYPE_PREPAID {
			_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
				disk.BillingType = BILLING_TYPE_POSTPAID
				disk.BillingCycle = ""
				disk.ExpiredAt = time.Time{}
				disk.StorageId = fakeStorage.Id
				return nil
			})
			if err != nil {
				log.Errorf("clear billing information for %d %s disk fail: %s", i, disk.DiskType, err)
				fakeHost.RealDelete(ctx, userCred)
				return err
			}
		}
	}

	return nil
}

func (self *SGuest) AllowPerformUndoPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "undo-prepaid-recycle")
}

func (self *SGuest) PerformUndoPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	host := self.GetHost()

	if host == nil {
		return nil, httperrors.NewInvalidStatusError("no valid host")
	}

	if host.Enabled {
		return nil, httperrors.NewInvalidStatusError("host should be disabled")
	}

	if host.ResourceType != HostResourceTypePrepaidRecycle || host.BillingType != BILLING_TYPE_PREPAID {
		return nil, httperrors.NewInvalidStatusError("host is not a prepaid recycle host")
	}

	err := doUndoPrepaidRecycle(ctx, userCred, host, self)
	if err != nil {
		logclient.AddActionLog(self, logclient.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}

	db.OpsLog.LogEvent(self, db.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred)
	logclient.AddActionLog(self, logclient.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred, true)

	return nil, nil
}

func (self *SHost) AllowPerformUndoPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "undo-prepaid-recycle")
}

func (self *SHost) PerformUndoPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		return nil, httperrors.NewInvalidStatusError("host should be disabled")
	}

	if self.ResourceType != HostResourceTypePrepaidRecycle || self.BillingType != BILLING_TYPE_PREPAID {
		return nil, httperrors.NewInvalidStatusError("host is not a prepaid recycle host")
	}

	guests := self.GetGuests()

	if len(guests) == 0 {
		return nil, httperrors.NewInvalidStatusError("cannot delete a recycle host without active instance")
	}

	if len(guests) > 1 {
		return nil, httperrors.NewInvalidStatusError("a recycle host shoud not allocate more than 1 guest")
	}

	if guests[0].PendingDeleted {
		return nil, httperrors.NewInvalidStatusError("cannot undo a recycle host with pending_deleted guest")
	}

	err := doUndoPrepaidRecycle(ctx, userCred, self, &guests[0])
	if err != nil {
		logclient.AddActionLog(self, logclient.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}

	db.OpsLog.LogEvent(self, db.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred)
	logclient.AddActionLog(self, logclient.ACT_UNDO_RECYCLE_PREPAID, self.GetShortDesc(), userCred, true)

	return nil, nil
}

func findIdiskById(idisks []cloudprovider.ICloudDisk, uuid string) cloudprovider.ICloudDisk {
	for i := 0; i < len(idisks); i += 1 {
		if idisks[i].GetGlobalId() == uuid {
			return idisks[i]
		}
	}
	return nil
}

func doUndoPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, server *SGuest) error {
	if host.RealExternalId != server.ExternalId {
		msg := "host and server external id not match!!!!"
		log.Errorf(msg)
		return errors.New(msg)
	}

	q := HostManager.Query()
	q = q.Equals("external_id", host.ExternalId)
	q = q.Equals("host_type", host.HostType)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("resource_type")),
		sqlchemy.Equals(q.Field("resource_type"), HostResourceTypeShared),
	))

	oHostCnt := q.Count()

	if oHostCnt == 0 {
		msg := "orthordox host not found???"
		log.Errorf(msg)
		return errors.New(msg)
	}
	if oHostCnt > 1 {
		msg := fmt.Sprintf("more than 1 (%d) orthordox host found???", oHostCnt)
		log.Errorf(msg)
		return errors.New(msg)
	}

	oHost := SHost{}
	oHost.SetModelManager(HostManager)

	err := q.First(&oHost)
	if err != nil {
		msg := fmt.Sprintf("fail to query orthordox host %s", err)
		log.Errorf(msg)
		return errors.New(msg)
	}

	iHost, err := oHost.GetIHost()
	if err != nil {
		msg := fmt.Sprint("fail to find ihost %s", err)
		log.Errorf(msg)
		return errors.New(msg)
	}

	iVM, err := iHost.GetIVMById(server.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to GetIVMById %s", err)
		log.Errorf(msg)
		return errors.New(msg)
	}

	idisks, err := iVM.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("iVM.GetIDisks fail %s", err)
		log.Errorf(msg)
		return errors.New(msg)
	}

	_, err = server.GetModelManager().TableSpec().Update(server, func() error {
		// recover billing information
		server.BillingType = BILLING_TYPE_PREPAID
		server.BillingCycle = host.BillingCycle
		server.ExpiredAt = host.ExpiredAt
		// switch to original Host
		server.HostId = oHost.Id
		return nil
	})
	if err != nil {
		log.Errorf("fail to recover vm hostId %s", err)
		return err
	}

	guestdisks := server.GetDisks()

	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		storage := disk.GetStorage()
		idisk := findIdiskById(idisks, disk.ExternalId)
		if idisk == nil {
			msg := fmt.Sprintf("fail to find idisk by ID %s: %s", disk.ExternalId, err)
			log.Errorf(msg)
			return errors.New(msg)
		}
		istorage, err := idisk.GetIStorage()
		if err != nil {
			log.Errorf("idisk.GetIStorage fail %s", err)
			return err
		}

		oHostStorage := oHost.GetHoststorageByExternalId(istorage.GetGlobalId())
		if oHostStorage == nil {
			msg := fmt.Sprintf("oHost.GetHoststorageByExternalId not found %s", istorage.GetGlobalId())
			log.Errorf(msg)
			return errors.New(msg)
		}

		oStorage := oHostStorage.GetStorage()

		if storage.StorageType == STORAGE_LOCAL {
			_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
				disk.BillingType = BILLING_TYPE_PREPAID
				disk.BillingCycle = host.BillingCycle
				disk.ExpiredAt = host.ExpiredAt
				disk.StorageId = oStorage.Id
				return nil
			})
			if err != nil {
				log.Errorf("fail to recover prepaid disk info %s", err)
				return err
			}
		}
	}

	err = host.RealDelete(ctx, userCred)
	if err != nil {
		log.Errorf("fail to delete fake host")
		return err
	}

	return nil
}

func (self *SGuest) IsPrepaidRecycle() bool {
	host := self.GetHost()
	if host == nil {
		return false
	}
	return host.IsPrepaidRecycle()
}

func (host *SHost) IsPrepaidRecycle() bool {
	if host.ResourceType != HostResourceTypePrepaidRecycle {
		return false
	}
	if host.BillingType != BILLING_TYPE_PREPAID {
		return false
	}
	return true
}

func (self *SHost) BorrowIpAddrsFromGuest(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	guestnics := guest.GetNetworks()
	for i := 0; i < len(guestnics); i += 1 {
		err := guestnics[i].Detach(ctx, userCred)
		if err != nil {
			log.Errorf("fail to detach guest network %s", err)
			return err
		}

		netif := self.GetNetInterface(guestnics[i].MacAddr)
		if netif == nil {
			msg := fmt.Sprintf("fail to find netinterface for mac %s", guestnics[i].MacAddr)
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}

		err = self.EnableNetif(ctx, userCred, netif, "", guestnics[i].IpAddr, "", false, false)
		if err != nil {
			log.Errorf("fail to enable netif %s %s", guestnics[i].IpAddr, err)
			return err
		}
	}
	return nil
}

func (host *SHost) SetGuestCreateNetworkAndDiskParams(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		log.Errorf("host.GetIHost fail %s", err)
		return nil, err
	}

	ivm, err := ihost.GetIVMById(host.RealExternalId)
	if err != nil {
		log.Errorf("ihost.GetIVMById(host.RealExternalId) fail %s", err)
		return nil, err
	}

	idisks, err := ivm.GetIDisks()
	if err != nil {
		log.Errorf("ivm.GetIDisks fail %s", err)
		return nil, err
	}

	netifs := host.GetNetInterfaces()
	netIdx := 0
	for i := 0; i < len(netifs); i += 1 {
		hn := netifs[i].GetBaremetalNetwork()
		if hn != nil {
			err := host.DisableNetif(ctx, userCred, &netifs[i], true)
			if err != nil {
				return nil, err
			}
			packedMac := strings.Replace(netifs[i].Mac, ":", "", -1)
			netDesc := fmt.Sprintf("%s:%s:%s:[reserved]", packedMac, hn.IpAddr, hn.NetworkId)
			params.Set(fmt.Sprintf("net.%d", netIdx), jsonutils.NewString(netDesc))
			netIdx += 1
		}
	}
	params.Set(fmt.Sprintf("net.%d", netIdx), jsonutils.JSONNull)

	for i := 0; i < len(idisks); i += 1 {
		/*istorage, err := idisks[i].GetIStorage()
		if err != nil {
			log.Errorf("idisks[i].GetIStorage fail %s", err)
			return nil, err
		}*/

		key := fmt.Sprintf("disk.%d", i)
		jsonConf, _ := params.Get(key)
		if jsonConf != nil && jsonConf != jsonutils.JSONNull {
			conf, err := parseDiskInfo(ctx, userCred, jsonConf)
			if err != nil {
				log.Debugf("parseDiskInfo %s fail %s", jsonConf, err)
				return nil, err
			}
			conf.SizeMb = idisks[i].GetDiskSizeMB()
			conf.Backend = STORAGE_LOCAL
			params.Set(key, jsonutils.Marshal(conf))
		} else {
			strConf := fmt.Sprintf("%s:%d", STORAGE_LOCAL, idisks[i].GetDiskSizeMB())
			conf, err := parseDiskInfo(ctx, userCred, jsonutils.NewString(strConf))
			if err != nil {
				return nil, err
			}
			params.Set(key, jsonutils.Marshal(conf))
		}
	}
	params.Set(fmt.Sprintf("disk.%d", len(idisks)), jsonutils.JSONNull)

	// log.Debugf("params after rebuid: %s", params.String())

	return params, nil
}

func (host *SHost) RebuildRecycledGuest(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	oHost := SHost{}
	oHost.SetModelManager(HostManager)

	q := HostManager.Query()
	q = q.Equals("external_id", host.ExternalId)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("resource_type")),
		sqlchemy.Equals(q.Field("resource_type"), HostResourceTypeShared),
	))

	err := q.First(&oHost)
	if err != nil {
		log.Errorf("query oHost fail %s", err)
		return err
	}

	err = guest.SetExternalId(host.RealExternalId)
	if err != nil {
		log.Errorf("guest.SetExternalId fail %s", err)
		return err
	}

	extVM, err := guest.GetIVM()
	if err != nil {
		log.Errorf("guest.GetIVM fail %s", err)
		return err
	}

	err = guest.syncWithCloudVM(ctx, userCred, &oHost, extVM, "", false)
	if err != nil {
		log.Errorf("guest.syncWithCloudVM fail %s", err)
		return err
	}

	idisks, err := extVM.GetIDisks()
	if err != nil {
		log.Errorf("extVM.GetIDisks fail %s", err)
		return err
	}

	guestdisks := guest.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		err = disk.SetExternalId(idisks[i].GetGlobalId())
		if err != nil {
			log.Errorf("disk.SetExternalId fail %s", err)
			return err
		}
		err = disk.syncWithCloudDisk(ctx, userCred, idisks[i], i, "", false)
		if err != nil {
			log.Errorf("disk.syncWithCloudDisk fail %s", err)
			return err
		}
	}

	return nil
}

func (manager *SHostManager) GetHostByRealExternalId(eid string) *SHost {
	q := manager.Query()
	q = q.Equals("real_external_id", eid)

	host := SHost{}
	host.SetModelManager(manager)

	err := q.First(&host)

	if err != nil {
		return nil
	}

	return &host
}
