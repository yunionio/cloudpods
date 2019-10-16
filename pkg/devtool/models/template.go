package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type SDevtoolTemplate struct {
	SVSCronjob
	Playbook *ansible.Playbook `length:"text" nullable:"false" create:"required" get:"user" update:"user"`
	// db.SStandaloneResourceBase
	db.SVirtualResourceBase
}

type SDevtoolTemplateManager struct {
	db.SVirtualResourceBaseManager
}

var (
	DevtoolTemplateManager *SDevtoolTemplateManager
)

func init() {
	// dt interface{}, tableName string, keyword string, keywordPlural string
	DevtoolTemplateManager = &SDevtoolTemplateManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDevtoolTemplate{},
			"devtool_templates_tbl",
			"devtool_template",
			"devtool_templates",
		),
	}
	DevtoolTemplateManager.SetVirtualObject(DevtoolTemplateManager)
	db.RegisterModelManager(DevtoolTemplateManager)
}

func (apb *SDevtoolTemplate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	log.Errorf("[(apb *SDevtoolTemplate) PostCreate] data: %+v", data)
	apb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

}
