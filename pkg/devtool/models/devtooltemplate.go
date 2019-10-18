package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/mcclient/options"

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
	db.SVirtualResourceBase
}

type SDevtoolTemplateManager struct {
	db.SVirtualResourceBaseManager
}

var (
	DevtoolTemplateManager *SDevtoolTemplateManager
)

func init() {
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

func getServerAttrs(ID string, s *mcclient.ClientSession) map[string]string {
	keys := []string{
		"hypervisor",
		"id",
		"ips",
		"name",
		"region_id",
		"zone_id",
	}
	params := make(map[string]string)
	result, err := modules.Servers.Get(s, ID, nil)
	if err != nil {
		log.Errorf("Error show server: %s", err)
		return nil
	}
	for _, key := range keys {
		value, _ := result.GetString(key)
		if key == "ips" {
			key = "ip"
			if strings.Contains(value, ",") {
				value = strings.Split(value, ",")[0]
			}
		}
		params["server_"+key] = value
	}

	return params
}

func getInfluxdbURL() (string, error) {

	// get service id
	s := auth.GetAdminSession(nil, "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("influxdb"), "search")
	result, err := modules.ServicesV3.List(s, query)
	if err != nil {
		return "", err
	}
	if len(result.Data) == 0 {
		return "", nil
	}
	service := result.Data[0]
	service_id, err := service.GetString("id")
	if err != nil {
		return "", err
	}

	// get endpoint
	query = jsonutils.NewDict()
	//
	//{
	//	"details": false,
	//	"filter.0": "service_id.equals(pig)",
	//	"filter.1": "interface.equals(public)",
	//	"limit": 10,
	//	"offset": 0
	//}
	query.Add(jsonutils.NewBool(false), "details")
	query.Add(jsonutils.NewString(fmt.Sprintf("service_id.equals(%s)", service_id)), "filter.0")
	query.Add(jsonutils.NewString("interface.equals(public)"), "filter.1")
	query.Add(jsonutils.NewInt(1), "limit")
	query.Add(jsonutils.NewInt(0), "offset")
	result, err = modules.EndpointsV3.List(s, query)
	if err != nil || len(result.Data) == 0 {
		return "", err
	}
	endpoint := result.Data[0]
	return endpoint.GetString("url")
}

func renderExtraVars(vars map[string]string) {

	for key, value := range vars {
		if key == "influxdb" && value == "INFLUXDB" {
			url, err := getInfluxdbURL()
			if err == nil {
				vars[key] = url
			} else {
				log.Errorf("template binding: get influxdb url error: %s", err)
			}
		}
	}
}

func deleteAnsiblePlaybook(id string, s *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	apb, err := modules.AnsiblePlaybooks.Get(s, id, nil)
	if err != nil {
		log.Errorf("[deleteAnsiblePlaybook] get Ansible playbook error %s", err)
		return apb, err
	}
	status, _ := apb.GetString("status")
	if status == models.AnsiblePlaybookStatusRunning {
		apb, err = modules.AnsiblePlaybooks.PerformAction(s, id, "stop", nil)
		if err != nil {
			log.Errorf("[deleteAnsiblePlaybook] stop Ansible playbook error %s", err)
			return apb, err
		}
	}

	apb, err = modules.AnsiblePlaybooks.Delete(s, id, nil)
	if err != nil {
		log.Errorf("[deleteAnsiblePlaybook] Delete Ansible playbook error %s", err)
		return apb, err
	}
	log.Debugf("ansible playbook %s has been deleted successfully.", id)
	return apb, nil
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

	attrs := getServerAttrs(ServerID, s)
	newPlaybookName := obj.Name + "-" + obj.Id[0:8] + "-" + ServerID[0:8]
	if len(newPlaybookName) > 32 {
		newPlaybookName = newPlaybookName[0:32]
	}

	playbook := obj.Playbook
	playbook.Inventory.Hosts[0].Name = ServerID

	for key, value := range attrs {
		playbook.Inventory.Hosts[0].Vars[key] = value
	}
	log.Errorf("before: playbook.Inventory.Hosts[0].Vars: %+v", playbook.Inventory.Hosts[0].Vars)

	renderExtraVars(playbook.Inventory.Hosts[0].Vars)
	log.Errorf("after: playbook.Inventory.Hosts[0].Vars: %+v", playbook.Inventory.Hosts[0].Vars)

	params := jsonutils.Marshal(&playbook)

	host, _ := params.Get("inventory")
	file, _ := params.Get("files")
	mod, _ := params.Get("modules")

	newAnsiblPlaybookParams := jsonutils.NewDict()
	newAnsiblPlaybookParams.Add(jsonutils.NewString(newPlaybookName), "name")
	newAnsiblPlaybookParams.Add(host, "playbook", "inventory")
	newAnsiblPlaybookParams.Add(file, "playbook", "files")
	newAnsiblPlaybookParams.Add(mod, "playbook", "modules")

	apb, err := modules.AnsiblePlaybooks.Create(s, newAnsiblPlaybookParams)
	if err != nil {
		log.Warningf("creating playbook Error...%+v", err)
		return nil, err
	}
	ansibleId, _ := apb.GetString("id")

	//get cronjob struct and create obj
	newCronjobName := obj.Name + "-" + obj.Id[0:8] + "-" + ansibleId[0:8]
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

func (obj *SDevtoolTemplate) PerformUnbind(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * stop and delete playbook
	// * stop and delete cronjob

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

	apb, err := deleteAnsiblePlaybook(newPlaybookName, s)
	if err != nil {
		return nil, err
	}
	ansibleId, _ := apb.GetString("id")

	//get cronjob struct and create obj
	newCronjobName := obj.Name + "-" + obj.Id[0:8] + "-" + ansibleId[0:8]
	if len(newCronjobName) > 32 {
		newCronjobName = newCronjobName[0:32]
	}

	_, err = modules.DevToolCronjobs.Delete(s, newCronjobName, nil)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (obj *SDevtoolTemplate) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	obj.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	log.Errorf("data to update: %+v", data)

	opt := options.DevtoolTemplateUpdateOptions{}
	data.Unmarshal(&opt)
	log.Errorf("data to update: %+v", opt)
	if !opt.Rebind {
		return
	}

	items := make([]SCronjob, 0)
	q := CronjobManager.Query().Equals("template_id", obj.Id)
	err := q.All(&items)
	if err != nil {
		log.Errorf("query error: %s", err)
	}
	for _, item := range items {
		log.Errorf("log: item %+v", item.ServerID)
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(item.ServerID), "server_id")
		obj.PerformUnbind(ctx, userCred, nil, params)
		obj.PerformBind(ctx, userCred, nil, params)
	}
}
