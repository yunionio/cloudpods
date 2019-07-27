package multicloud

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDBInstanceBackupBase struct {
	SResourceBase
}

func (backup *SDBInstanceBackupBase) GetBackMode() string {
	return api.BACKUP_MODE_AUTOMATED
}
