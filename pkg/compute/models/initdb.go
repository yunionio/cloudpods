package models

import (
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/log"
)

func InitDB() error {
	for _, manager := range []db.IModelManager{
		CloudregionManager,
		ZoneManager,
		VpcManager,
		WireManager,
		StorageManager,
		SecurityGroupManager,
		NetworkManager,
	} {
		err := manager.InitializeData()
		if err != nil {
			log.Errorf("Manager %s initializeData fail %s", manager.Keyword(), err)
			return err
		}
	}
	return nil
}
