package models

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func InitDB() error {
	for _, manager := range []db.IModelManager{
		// DifyManager,
		// LLMManager,
	} {
		err := manager.InitializeData()
		if err != nil {
			log.Errorf("Manager %s initializeData fail %s", manager.Keyword(), err)
			return err
		} else {
			log.Infof("Manager %s initializeData PASS!", manager.Keyword())
		}
	}
	return nil
}
