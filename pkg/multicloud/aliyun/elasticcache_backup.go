package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheBackup struct {
	multicloud.SElasticcacheBackupBase

	cacheDB *SElasticcache

	BackupIntranetDownloadURL string    `json:"BackupIntranetDownloadURL"`
	BackupType                string    `json:"BackupType"`
	BackupEndTime             time.Time `json:"BackupEndTime"`
	BackupMethod              string    `json:"BackupMethod"`
	BackupID                  int64     `json:"BackupId"`
	BackupStartTime           time.Time `json:"BackupStartTime"`
	BackupDownloadURL         string    `json:"BackupDownloadURL"`
	BackupDBNames             string    `json:"BackupDBNames"`
	NodeInstanceID            string    `json:"NodeInstanceId"`
	BackupMode                string    `json:"BackupMode"`
	BackupStatus              string    `json:"BackupStatus"`
	BackupSizeByte            int64     `json:"BackupSize"`
	EngineVersion             string    `json:"EngineVersion"`
}

func (self *SElasticcacheBackup) GetId() string {
	return fmt.Sprintf("%d", self.BackupID)
}

func (self *SElasticcacheBackup) GetName() string {
	return self.GetId()
}

func (self *SElasticcacheBackup) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheBackup) GetStatus() string {
	return ""
}

func (self *SElasticcacheBackup) GetBackupSizeMb() int {
	return int(self.BackupSizeByte / 1024 / 1024)
}

func (self *SElasticcacheBackup) GetBackupType() string {
	return self.BackupType
}

func (self *SElasticcacheBackup) GetBackupMode() string {
	return self.BackupMode
}

func (self *SElasticcacheBackup) GetDownloadURL() string {
	return self.BackupDownloadURL
}

func (self *SElasticcacheBackup) GetStartTime() time.Time {
	return self.BackupStartTime
}

func (self *SElasticcacheBackup) GetEndTime() time.Time {
	return self.BackupEndTime
}
