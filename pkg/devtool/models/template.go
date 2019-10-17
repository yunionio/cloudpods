package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/ansibleserver/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

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

func deleteAnsiblePlaybook(id string, s *mcclient.ClientSession) error {
	apb, err := modules.AnsiblePlaybooks.Get(s, id, nil)
	if err != nil {
		log.Errorf("[deleteAnsiblePlaybook] get Ansible playbook error %s", err)
		return err
	}
	status, _ := apb.GetString("status")
	if status == models.AnsiblePlaybookStatusRunning {
		apb, err = modules.AnsiblePlaybooks.PerformAction(s, id, "stop", nil)
		if err != nil {
			log.Errorf("[deleteAnsiblePlaybook] stop Ansible playbook error %s", err)
			return err
		}
	}

	_, err = modules.AnsiblePlaybooks.Delete(s, id, nil)
	if err != nil {
		log.Errorf("[deleteAnsiblePlaybook] Delete Ansible playbook error %s", err)
		return err
	}
	log.Debugf("ansible playbook %s has been deleted successfully.", id)
	return nil
}

func (obj *SDevtoolTemplate) PerformBind(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * get playbook struct and create obj
	// * get cronjob struct and create obj
	// * create playbook

	s := auth.GetSession(ctx, userCred, "", "")
	ServerID, err := data.GetString("server_id")
	if err != nil {
		log.Errorf("Get server ID error: %s", err)
		return nil, err
	}

	newPlaybookName := obj.Name + "-" + obj.Id[0:8] + "-" + ServerID[0:8]
	if len(newPlaybookName) > 32 {
		newPlaybookName = newPlaybookName[0:32]
	}

	playbook := obj.Playbook
	playbook.Inventory.Hosts[0].Name = ServerID
	params := jsonutils.Marshal(&playbook)

	host, _ := params.Get("inventory")
	file, _ := params.Get("files")
	mod, _ := params.Get("modules")

	newAnsiblPlaybookParams := jsonutils.NewDict()
	newAnsiblPlaybookParams.Add(jsonutils.NewString(newPlaybookName), "name")
	newAnsiblPlaybookParams.Add(host, "playbook", "host")
	newAnsiblPlaybookParams.Add(file, "playbook", "file")
	newAnsiblPlaybookParams.Add(mod, "playbook", "mod")

	apb, err := modules.AnsiblePlaybooks.Create(s, newAnsiblPlaybookParams)
	if err != nil {
		log.Warningf("creating playbook Error...%+v", err)
		return nil, err
	}
	ansibleId, _ := apb.GetString("id")
	err = deleteAnsiblePlaybook(ansibleId, s)
	if err != nil {
		return nil, err
	}

	//get cronjob struct and create obj
	newCronjobName := obj.Name + "-" + obj.Id[0:8] + ansibleId[0:8]
	if len(newCronjobName) > 32 {
		newCronjobName = newCronjobName[0:32]
	}

	newCronjobParams := jsonutils.NewDict()
	newCronjobParams.Add(jsonutils.NewString(newCronjobName), "name")
	newCronjobParams.Add(jsonutils.NewInt(int64(obj.Day)), "day")
	newCronjobParams.Add(jsonutils.NewInt(int64(obj.Hour)), "hour")
	newCronjobParams.Add(jsonutils.NewInt(int64(obj.Min)), "min")
	newCronjobParams.Add(jsonutils.NewInt(int64(obj.Sec)), "sec")
	newCronjobParams.Add(jsonutils.NewInt(int64(obj.Interval)), "interval")
	newCronjobParams.Add(jsonutils.NewBool(obj.Start), "start")
	newCronjobParams.Add(jsonutils.NewBool(obj.Enabled), "enabled")
	newCronjobParams.Add(jsonutils.NewString(ansibleId), "ansible_playbook_id")
	newCronjobParams.Add(jsonutils.NewString(obj.Id), "template_id")
	newCronjobParams.Add(jsonutils.NewString(ServerID), "server_id")

	_, err = modules.DevToolCronjobs.Create(s, newCronjobParams)
	if err != nil {
		log.Infof("modules.DevToolCronjobs.Create error %s", err)
		return nil, err
	}

	return nil, nil
}
