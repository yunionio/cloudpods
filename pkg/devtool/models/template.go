package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type SDevtoolTemplate struct {
	SVSCronjob
	Playbook *ansible.Playbook
	db.SStandaloneResourceBase
}

type SDevtoolTemplateManager struct {
	db.SStandaloneResourceBaseManager
}

var (
	DevtoolTemplateManager *SDevtoolTemplateManager
)

func init() {
	// dt interface{}, tableName string, keyword string, keywordPlural string
	DevtoolTemplateManager = &SDevtoolTemplateManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDevtoolTemplate{},
			"devtool_templates_tbl",
			"devtool_template",
			"devtool_templates",
		),
	}
	DevtoolTemplateManager.SetVirtualObject(DevtoolTemplateManager)
	db.RegisterModelManager(DevtoolTemplateManager)
}
