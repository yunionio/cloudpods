package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var PersistentVolumeClaims *PersistentVolumeClaimManager

type PersistentVolumeClaimManager struct {
	*NamespaceResourceManager
	statusGetter
}

func init() {
	PersistentVolumeClaims = &PersistentVolumeClaimManager{
		NamespaceResourceManager: NewNamespaceResourceManager(
			"persistentvolumeclaim", "persistentvolumeclaims",
			NewColumns("Status", "Volume", "StorageClass"), NewColumns()),
		statusGetter: getStatus,
	}
	modules.Register(PersistentVolumeClaims)
}

func (m PersistentVolumeClaimManager) GetVolume(obj jsonutils.JSONObject) interface{} {
	volume, _ := obj.GetString("volume")
	return volume
}

func (m PersistentVolumeClaimManager) GetStorageClass(obj jsonutils.JSONObject) interface{} {
	sc, _ := obj.GetString("storageClass")
	return sc
}
