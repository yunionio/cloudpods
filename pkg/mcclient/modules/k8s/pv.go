package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var PersistentVolumes *PersistentVolumeManager

type PersistentVolumeManager struct {
	*MetaResourceManager
}

func init() {
	PersistentVolumes = &PersistentVolumeManager{
		MetaResourceManager: NewMetaResourceManager("persistentvolume", "persistentvolumes",
			NewColumns("StorageClass", "Claim", "AccessModes"),
			NewColumns()),
	}

	modules.Register(PersistentVolumes)
}

func (m PersistentVolumeManager) GetStorageClass(obj jsonutils.JSONObject) interface{} {
	sc, _ := obj.GetString("storageClass")
	return sc
}

func (m PersistentVolumeManager) GetClaim(obj jsonutils.JSONObject) interface{} {
	claim, _ := obj.GetString("claim")
	return claim
}

func (m PersistentVolumeManager) GetAccessModes(obj jsonutils.JSONObject) interface{} {
	modes, _ := obj.(*jsonutils.JSONDict).GetArray("accessModes")
	return modes
}
