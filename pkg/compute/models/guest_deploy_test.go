// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"testing"

	"yunion.io/x/jsonutils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestValidateDeployConfigs(t *testing.T) {
	cfgs := []*api.DeployConfig{
		{Action: "create", Path: "/etc/hosts", Content: "x"},
	}
	if err := ValidateDeployConfigs(cfgs); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if cfgs[0].Path != "/etc/hosts" {
		t.Fatalf("path %q", cfgs[0].Path)
	}

	rel := []*api.DeployConfig{{Action: "create", Path: "attack/etc/pwn"}}
	if err := ValidateDeployConfigs(rel); err == nil {
		t.Fatal("expected relative path to fail")
	}
	root := []*api.DeployConfig{{Action: "create", Path: "/"}}
	if err := ValidateDeployConfigs(root); err == nil {
		t.Fatal("expected / to fail")
	}
	badAct := []*api.DeployConfig{{Action: "exec", Path: "/etc/hosts"}}
	if err := ValidateDeployConfigs(badAct); err == nil {
		t.Fatal("expected invalid action to fail")
	}
	cleaned := []*api.DeployConfig{{Path: "/etc/../root/.ssh/authorized_keys"}}
	if err := ValidateDeployConfigs(cleaned); err != nil {
		t.Fatalf("cleaned path: %v", err)
	}
	if cleaned[0].Path != "/root/.ssh/authorized_keys" {
		t.Fatalf("got %q", cleaned[0].Path)
	}
}

func TestNewGuestDeployTaskParam(t *testing.T) {
	param, err := NewGuestDeployTaskParam(nil)
	if err != nil || param != nil {
		t.Fatalf("nil data: param=%v err=%v", param, err)
	}

	input := api.ServerDeployInput{}
	input.ResetPassword = false
	input.KeypairId = "0123456789abcdef0123456789abcdef"
	input.Password = "secret"
	input.Restart = true
	param, err = NewGuestDeployTaskParam(jsonutils.Marshal(input))
	if err != nil {
		t.Fatalf("deploy input: %v", err)
	}
	if param.ResetPassword == nil || *param.ResetPassword {
		t.Fatalf("reset_password %#v", param.ResetPassword)
	}
	if param.Keypair != input.KeypairId || param.KeypairId != input.KeypairId {
		t.Fatalf("keypair %q id %q", param.Keypair, param.KeypairId)
	}
	param.DeployAction = api.TDeployActionDeploy
	data := jsonutils.Marshal(param).(*jsonutils.JSONDict)
	if jsonutils.QueryBoolean(data, "reset_password", true) {
		t.Fatal("marshaled reset_password should be false")
	}
	keypair, _ := data.GetString("keypair")
	if keypair != input.KeypairId {
		t.Fatalf("marshaled keypair %q", keypair)
	}
	if !jsonutils.QueryBoolean(data, "restart", false) {
		t.Fatal("restart should be true")
	}

	param, err = NewGuestDeployTaskParam(jsonutils.NewDict())
	if err != nil {
		t.Fatal(err)
	}
	if param.ResetPassword != nil {
		t.Fatal("unspecified reset_password should stay nil")
	}
	data = jsonutils.Marshal(param).(*jsonutils.JSONDict)
	if data.Contains("reset_password") {
		t.Fatal("nil reset_password should be omitted")
	}
	if !jsonutils.QueryBoolean(data, "reset_password", true) {
		t.Fatal("missing reset_password should default to true")
	}

	createParams := jsonutils.NewDict()
	createParams.Set("vcpu_count", jsonutils.NewInt(2))
	createParams.Set("password", jsonutils.NewString("pw"))
	createParams.Set("enable_cloud_init", jsonutils.JSONTrue)
	createParams.Set("public_ip_bw", jsonutils.NewInt(10))
	createParams.Set("public_ip_charge_type", jsonutils.NewString(string(billing_api.NET_CHARGE_TYPE_BY_TRAFFIC)))
	createParams.Set("auto_start", jsonutils.JSONTrue)
	param, err = NewGuestDeployTaskParam(createParams)
	if err != nil {
		t.Fatalf("create params: %v", err)
	}
	if param.Password != "pw" || !param.EnableCloudInit || !param.AutoStart {
		t.Fatalf("create param %+v", param)
	}
	if param.PublicIpBw != 10 || param.PublicIpChargeType != billing_api.NET_CHARGE_TYPE_BY_TRAFFIC {
		t.Fatalf("public ip %+v %q", param.PublicIpBw, param.PublicIpChargeType)
	}
	if param.ResetPassword != nil {
		t.Fatal("create params without reset_password should stay nil")
	}
}

func TestGuestDeployConfigJSONShape(t *testing.T) {
	reset := false
	cfg := &api.SGuestDeployConfigOnHost{
		Desc:             jsonutils.Marshal(map[string]string{"name": "vm"}),
		Action:           api.TDeployActionDeploy,
		OnFinish:         "shutdown",
		EnableCloudInit:  false,
		ResetPassword:    &reset,
		AdminPublicKey:   "admin",
		ProjectPublicKey: "proj",
		Deploys: []*api.DeployConfig{
			{Action: "create", Path: "/etc/hosts", Content: "x"},
		},
	}
	data := jsonutils.Marshal(cfg).(*jsonutils.JSONDict)
	name, err := data.GetString("desc", "name")
	if err != nil || name != "vm" {
		t.Fatalf("desc name: %v %q", err, name)
	}
	if action, _ := data.GetString("action"); action != "deploy" {
		t.Fatalf("action %q", action)
	}
	if jsonutils.QueryBoolean(data, "reset_password", true) {
		t.Fatal("reset_password should be false")
	}
	if jsonutils.QueryBoolean(data, "enable_cloud_init", true) {
		t.Fatal("enable_cloud_init should be false")
	}
	if jsonutils.QueryBoolean(data, "deploy_telegraf", true) {
		t.Fatal("deploy_telegraf should be false")
	}
	if jsonutils.QueryBoolean(data, "disable_image_cache", true) || jsonutils.QueryBoolean(data, "fake_create_from_bm_import", true) {
		t.Fatalf("bool flags should be false: %s", data.String())
	}
	if !data.Contains("admin_public_key") || !data.Contains("deploys") || !data.Contains("project_public_key") {
		t.Fatalf("host fields missing: %s", data.String())
	}
	if data.Contains("password") || data.Contains("restart") || data.Contains("deploy_action") || data.Contains("deploy_configs") {
		t.Fatalf("unexpected fields: %s", data.String())
	}
}

func TestDeployGuestFsResponseJSONShape(t *testing.T) {
	resp := &api.SDeployGuestFsResponse{
		Os:               "Linux",
		Account:          "root",
		Key:              "secret",
		Distro:           "CentOS",
		Version:          "7.9",
		Arch:             "x86_64",
		Language:         "en_US",
		TelegrafDeployed: true,
		CurrentVersion:   "1",
		Uuid:             "vm-1",
		HostIp:           "10.0.0.2",
		AutoRenew:        true,
		Disks: []api.SDeployGuestFsDisk{
			{DiskId: "disk-1", Size: 40960, PciPath: "pci", Dev: "sda"},
		},
		Metadata: map[string]string{"os": "linux"},
	}
	data := jsonutils.Marshal(resp).(*jsonutils.JSONDict)
	got := api.SDeployGuestFsResponse{}
	if err := data.Unmarshal(&got); err != nil {
		t.Fatal(err)
	}
	if got.Os != "Linux" || got.Account != "root" || got.Key != "secret" || got.Distro != "CentOS" {
		t.Fatalf("fs fields %+v", got)
	}
	if !got.TelegrafDeployed || !got.AutoRenew || got.Uuid != "vm-1" || got.HostIp != "10.0.0.2" {
		t.Fatalf("callback fields %+v", got)
	}
	if len(got.Disks) != 1 || got.Disks[0].DiskId != "disk-1" || got.Disks[0].Size != 40960 || got.Disks[0].Dev != "sda" {
		t.Fatalf("disks %+v", got.Disks)
	}
	if got.Metadata["os"] != "linux" {
		t.Fatalf("metadata %+v", got.Metadata)
	}

	empty := jsonutils.Marshal(&api.SDeployGuestFsResponse{}).(*jsonutils.JSONDict)
	if empty.Contains("os") || empty.Contains("disks") || empty.Contains("uuid") || empty.Contains("expired_at") {
		t.Fatalf("empty response should omit zero strings and slices: %s", empty.String())
	}
	if !empty.Contains("telegraf_deployed") || jsonutils.QueryBoolean(empty, "telegraf_deployed", true) {
		t.Fatalf("telegraf_deployed false should be kept: %s", empty.String())
	}
	if !empty.Contains("auto_renew") || jsonutils.QueryBoolean(empty, "auto_renew", true) {
		t.Fatalf("auto_renew false should be kept: %s", empty.String())
	}
}
