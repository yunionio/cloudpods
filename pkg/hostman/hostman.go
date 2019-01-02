package hostman

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type IGuestManager interface {
	GetGuestNicDesc(mac, ip, port, bridge string, isCandidate bool) (jsonutils.JSONObject, jsonutils.JSONObject)
}

type IHostInfo interface {
}

type IStorageManager interface {
	GetStorageDisk(storageId, diskId string) storageman.IDisk
	GetDiskByPath(diskPath string) storageman.IDisk

	GetStorage(storageId string) Istorage
}

type IIsolatedDeviceManager interface {
}

var (
	hostInstance          IHostInfo
	guestManager          IGuestManager
	storageManager        IStorageManager
	isolatedDeviceManager IIsolatedDeviceManager
)

func HostInstance() IHostInfo {
	return hostInstance
}

func GuestManager() IGuestManager {
	return guestManager
}

func StorageManager() IStorageManager {
	return storageManager
}

func IsolatedDeviceManager() IIsolatedDeviceManager {
	return isolatedDeviceManager
}

func Init(h IHostInfo, g IGuestManager, s IStorageManager, i IIsolatedDeviceManager) {
	log.Infof("Hostman Init Managers ......")
	hostInstance = h
	guestManager = g
	storageManager = s
	isolatedDeviceManager = i
}
