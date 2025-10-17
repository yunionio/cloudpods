package models

import (
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var accessInfoManager *SAccessInfoManager

func init() {
	GetAccessInfoManager()
}

func GetAccessInfoManager() *SAccessInfoManager {
	if accessInfoManager == nil {
		accessInfoManager = &SAccessInfoManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SAccessInfo{},
				"access_infos_tbl",
				"access_info",
				"access_infos",
			),
		}
		accessInfoManager.SetVirtualObject(accessInfoManager)
	}
	return accessInfoManager
}

type SAccessInfoManager struct {
	db.SResourceBaseManager
}

type SAccessInfo struct {
	db.SResourceBase

	LLMId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`

	// 服务监听端口
	ListenPort int `nullable:"true" create:"optional" list:"user" update:"user"`
	// 映射到公网的访问端口
	AccessPort int `nullable:"true" create:"optional" list:"user" update:"user"`

	// 自定义端口类型
	Protocol string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`

	RemoteIps       []string            `charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	PortMappingEnvs api.PortMappingEnvs `charset:"ascii" nullable:"true" list:"user" create:"admin_optional"`
}
