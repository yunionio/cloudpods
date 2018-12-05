package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SGuest) PerformPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.BillingType != BILLING_TYPE_PREPAID {
		return nil, httperrors.NewInvalidStatusError("recycle prepaid server only")
	}
	if self.ExpiredAt.Before(time.Now()) {
		return nil, httperrors.NewInvalidStatusError("prepaid expired")
	}

	err := self.doPrepaidRecycle(ctx, userCred)
	if err != nil {
		return nil, err
	}

	return nil, self.doPrepaidRecycle(ctx, userCred)
}

func (self *SGuest) doPrepaidRecycle(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockClass(ctx, HostManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, HostManager, userCred.GetProjectId())

	return self.doPrepaidRecycleNoLock(ctx, userCred)
}

func (self *SGuest) doPrepaidRecycleNoLock(ctx context.Context, userCred mcclient.TokenCredential) error {
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
		info.Driver = storage.StorageType
		info.Rotate = (storage.MediumType != DISK_TYPE_SSD)

		storageInfo = append(storageInfo, info)
	}

	fakeHost.StorageDriver = baremetal.DISK_DRIVER_LINUX
	fakeHost.StorageSize = totalSize
	fakeHost.StorageInfo = jsonutils.Marshal(&storageInfo)

	fakeHost.ZoneId = self.getZone().GetId()
	fakeHost.IsBaremetal = false
	fakeHost.IsMaintenance = false
	fakeHost.HostType = self.GetHostType()
	fakeHost.ResourceType = HostResourceTypePrepaidRecycle

	guestnics := self.GetNetworks()
	if len(guestnics) == 0 {
		msg := "no network info on guest????"
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}
	fakeHost.AccessIp = guestnics[0].IpAddr
	fakeHost.AccessMac = guestnics[0].MacAddr

	fakeHost.BillingType = self.BillingType
	fakeHost.BillingCycle = self.BillingCycle
	fakeHost.ExpiredAt = self.ExpiredAt

	fakeHost.Status = HOST_STATUS_RUNNING
	fakeHost.Enabled = true
	fakeHost.ExternalId = self.ExternalId
	fakeHost.ManagerId = self.GetHost().ManagerId
	fakeHost.IsEmulated = true
	fakeHost.Description = "fake host for prepaid vm recycling"

	err := HostManager.TableSpec().Insert(&fakeHost)
	if err != nil {
		log.Errorf("fail to insert fake host %s", err)
		return err
	}

	log.Infof("save fakeHost success %s", fakeHost.Id)

	for i := 0; i < len(guestnics); i += 1 {
		err = fakeHost.addNetif(ctx, userCred,
			guestnics[i].MacAddr,
			guestnics[i].GetNetwork().WireId,
			"",
			1000,
			"",
			int8(i),
			tristate.True,
			1500,
			false,
			"",
			"",
			false,
			false)
		if err != nil {
			log.Errorf("fail to addNetInterface %d: %s", i, err)
			fakeHost.RealDelete(ctx, userCred)
			return err
		}
	}

	for i := 0; i < len(guestdisks); i += 1 {
		err = fakeHost.Attach2Storage(ctx, userCred, guestdisks[i].GetDisk().GetStorage(), "")
		if err != nil {
			log.Errorf("fail to addStorage %d: %s", i, err)
			fakeHost.RealDelete(ctx, userCred)
			return err
		}
	}

	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.HostId = fakeHost.Id
		return nil
	})

	if err != nil {
		log.Errorf("fail to change vm hostId", err)
		fakeHost.RealDelete(ctx, userCred)
		return err
	}

	return nil
}
