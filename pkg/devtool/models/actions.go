// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	apis "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func getServerAttrs(ID string, s *mcclient.ClientSession) (map[string]string, error) {
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
		return nil, err
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
	return params, err
}

func getInfluxdbURL() (string, error) {

	s := auth.GetAdminSessionWithPublic(nil, "", "")
	url, err := s.GetServiceURL("influxdb", auth.PublicEndpointType)

	if err != nil {
		log.Errorf("get influxdb Endpoint error %s", err)
		return "", err
	}
	return url, nil
}

func renderExtraVars(vars map[string]string) {

	InfluxdbURL, err := getInfluxdbURL()
	if err != nil {
		log.Errorf("template binding: get influxdb url error: %s", err)
		return
	}
	for key, value := range vars {
		if key == "influxdb" && value == "INFLUXDB" {
			vars[key] = InfluxdbURL
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
	if status == apis.AnsiblePlaybookStatusRunning {
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

func (obj *SDevtoolTemplate) Binding(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * get playbook struct and create obj
	// * get cronjob struct and create obj
	// * create playbook

	template := obj
	s := auth.GetSession(ctx, userCred, "", "")
	ServerID, err := data.GetString("server_id")

	attrs, err := getServerAttrs(ServerID, s)
	if err != nil {
		log.Errorf("TemplateBindingServers getServerAttrs failed %s", err)
		return nil, err
	}
	newPlaybookName := template.Name + "-" + template.Id[0:8] + "-" + ServerID[0:8]
	if len(newPlaybookName) > 32 {
		newPlaybookName = newPlaybookName[0:32]
	}

	playbook := template.Playbook
	playbook.Inventory.Hosts[0].Name = ServerID

	for key, value := range attrs {
		playbook.Inventory.Hosts[0].Vars[key] = value
	}

	renderExtraVars(playbook.Inventory.Hosts[0].Vars)
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
		log.Errorf("TemplateBindingServers AnsiblePlaybooks.Create failed %s", err)
		return nil, err
	}
	ansibleId, _ := apb.GetString("id")

	//get cronjob struct and create template
	newCronjobName := template.Name + "-" + template.Id[0:8] + "-" + ansibleId[0:8]
	if len(newCronjobName) > 32 {
		newCronjobName = newCronjobName[0:32]
	}

	newCronjobParams := jsonutils.NewDict()
	newCronjobParams.Add(jsonutils.NewString(newCronjobName), "name")
	newCronjobParams.Add(jsonutils.NewInt(int64(template.Day)), "day")
	newCronjobParams.Add(jsonutils.NewInt(int64(template.Hour)), "hour")
	newCronjobParams.Add(jsonutils.NewInt(int64(template.Min)), "min")
	newCronjobParams.Add(jsonutils.NewInt(int64(template.Sec)), "sec")
	newCronjobParams.Add(jsonutils.NewInt(int64(template.Interval)), "interval")
	newCronjobParams.Add(jsonutils.NewBool(template.Start), "start")
	newCronjobParams.Add(jsonutils.NewBool(template.Enabled), "enabled")
	newCronjobParams.Add(jsonutils.NewString(ansibleId), "ansible_playbook_id")
	newCronjobParams.Add(jsonutils.NewString(template.Id), "template_id")
	newCronjobParams.Add(jsonutils.NewString(ServerID), "server_id")

	_, err = modules.DevToolCronjobs.Create(s, newCronjobParams)
	if err != nil {
		log.Errorf("TemplateBindingServers failed %s", err)
		return nil, err
	}
	return nil, nil
}

func (obj *SDevtoolTemplate) Unbinding(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// * get server id
	// * get playbook struct and create obj
	// * get cronjob struct and create obj
	// * create playbook
	template := obj
	s := auth.GetSession(ctx, userCred, "", "")
	ServerID, err := data.GetString("server_id")

	newPlaybookName := template.Name + "-" + template.Id[0:8] + "-" + ServerID[0:8]
	if len(newPlaybookName) > 32 {
		newPlaybookName = newPlaybookName[0:32]
	}

	apb, err := deleteAnsiblePlaybook(newPlaybookName, s)
	if err != nil {
		log.Errorf("TemplateUnbindingServers failed %s", err)
		return nil, err
	}
	ansibleId, _ := apb.GetString("id")
	newCronjobName := template.Name + "-" + template.Id[0:8] + "-" + ansibleId[0:8]

	if len(newCronjobName) > 32 {
		newCronjobName = newCronjobName[0:32]
	}

	_, err = modules.DevToolCronjobs.Delete(s, newCronjobName, nil)
	if err != nil {
		log.Errorf("err: %+v", err)
		log.Errorf("TemplateUnbindingServers failed %s", err)
		return nil, err
	}
	return nil, nil
}

func (obj *SDevtoolTemplate) TaskUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	template := obj
	template.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)

	opt := options.DevtoolTemplateUpdateOptions{}
	data.Unmarshal(&opt)
	if !opt.Rebind {
		return nil, nil
	}

	items := make([]SCronjob, 0)
	q := CronjobManager.Query().Equals("template_id", template.Id)
	err := q.All(&items)
	if err != nil {
		log.Errorf("query error: %s", err)
		return nil, err
	}
	for _, item := range items {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(item.ServerID), "server_id")
		template.Unbinding(ctx, userCred, nil, params)
		template.Binding(ctx, userCred, nil, params)
	}
	return nil, nil
}
