package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Storageclass *StorageclassManager

type StorageclassManager struct {
	*MetaResourceManager
}

func init() {
	Storageclass = &StorageclassManager{
		MetaResourceManager: NewMetaResourceManager("storageclass", "storageclasses", NewColumns("Provisioner"), NewColumns()),
	}

	modules.Register(Storageclass)
}

func (m StorageclassManager) GetProvisioner(obj jsonutils.JSONObject) interface{} {
	provisioner, _ := obj.GetString("provisioner")
	return provisioner
}
