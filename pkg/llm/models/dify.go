package models

import (
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var difyManager *SDifyManager

func init() {
	GetDifyManager()
}

func GetDifyManager() *SDifyManager {
	if difyManager != nil {
		return difyManager
	}
	difyManager = &SDifyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDify{},
			"difies_tbl",
			"dify",
			"difies",
		),
	}
	difyManager.SetVirtualObject(difyManager)
	return difyManager
}

type SDifyManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SDify struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	DifyModelId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`

	SvrId  string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	DifyIp string `width:"20" charset:"ascii" nullable:"true" list:"user"`
	// Hypervisor     string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	Priority    int `nullable:"false" default:"100" list:"user"`
	BandwidthMb int `nullable:"true" list:"user" create:"admin_optional"`

	LastAppProbe time.Time `nullable:"true" list:"user" create:"admin_optional"`

	// 是否请求同步更新镜像
	SyncImageRequest bool `default:"false" nullable:"false" list:"user" update:"user"`

	VolumeUsedMb int       `nullable:"true" list:"user"`
	VolumeUsedAt time.Time `nullable:"true" list:"user"`

	// 秒装应用配额（可安装的总容量限制）
	// InstantAppQuotaGb int `list:"user" update:"user" create:"optional" default:"0" nullable:"false"`

	DebugMode     bool `default:"false" nullable:"false" list:"user" update:"user"`
	RootfsUnlimit bool `default:"false" nullable:"false" list:"user" update:"user"`
}
