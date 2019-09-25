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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	ansible_apis "yunion.io/x/onecloud/pkg/apis/ansible"
	compute_apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_models "yunion.io/x/onecloud/pkg/mcclient/models"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SLoadbalancerAgentDeployment struct {
	Host                        string
	AnsiblePlaybook             string
	AnsiblePlaybookUndeployment string
}

func (p *SLoadbalancerAgentDeployment) String() string {
	return jsonutils.Marshal(p).String()
}

func (p *SLoadbalancerAgentDeployment) IsZero() bool {
	if *p == (SLoadbalancerAgentDeployment{}) {
		return true
	}
	return false
}

func (lbagent *SLoadbalancerAgent) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowPerform(userCred, lbagent, "deploy")
}

func (lbagent *SLoadbalancerAgent) AllowPerformUndeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowPerform(userCred, lbagent, "undeploy")
}

func (lbagent *SLoadbalancerAgent) deploy(ctx context.Context, userCred mcclient.TokenCredential, input *compute_apis.LoadbalancerAgentDeployInput) (*ansible.Playbook, error) {
	pb := &ansible.Playbook{
		Inventory: ansible.Inventory{
			Hosts: []ansible.Host{input.Host},
		},
		Modules: []ansible.Module{
			{
				Name: "group",
				Args: []string{
					"name=yunion",
					"state=present",
				},
			},
			{
				Name: "user",
				Args: []string{
					"name=yunion",
					"state=present",
					"group=yunion",
				},
			},
			{
				Name: "file",
				Args: []string{
					"path=/etc/yunion",
					"state=directory",
					"owner=yunion",
					"group=yunion",
					"mode=755",
				},
			},
			{
				Name: "template",
				Args: []string{
					"src=lbagentConfTmpl",
					"dest=/etc/yunion/lbagent.conf",
					"owner=yunion",
					"group=yunion",
					"mode=600",
				},
			},
		},
		Files: map[string][]byte{
			"lbagentConfTmpl": []byte(lbagentConfTmpl),
		},
	}
	switch input.DeployMethod {
	case compute_apis.DeployMethodYum:
		if v, ok := input.Host.GetVar("repo_base_url"); !ok || v == "" {
			return nil, httperrors.NewBadRequestError("use yum requires valid repo_base_url")
		}
		if v, ok := input.Host.GetVar("repo_sslverify"); !ok || v == "" {
			input.Host.SetVar("repo_sslverify", "0")
		}
		pb.Files["yunionRepoTmpl"] = []byte(yunionRepoTmpl)
		pb.Modules = append(pb.Modules,
			ansible.Module{
				Name: "template",
				Args: []string{
					"src=yunionRepoTmpl",
					"dest=/etc/yum.repos.d/yunion.repo",
					"owner=root",
					"group=root",
					"mode=644",
				},
			},
			ansible.Module{
				Name: "yum",
				Args: []string{
					"name=yunion-lbagent",
					"state=latest",
					"update_cache=yes",
				},
			},
		)
	case compute_apis.DeployMethodCopy:
		fallthrough
	default:
		// glob for rpms
		basenames := []string{
			"packages/telegraf",
			"packages/gobetween",
			"packages/keepalived",
			"packages/haproxy",
			"updates/yunion-lbagent",
		}
		mods := []ansible.Module{}
		for _, basename := range basenames {
			pattern := filepath.Join("/opt/yunion/upgrade/rpms", basename+"-*.rpm")
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, errors.WithMessagef(err, "glob error %s", pattern)
			}
			if len(matches) == 0 {
				return nil, errors.WithMessagef(err, "glob nomatch %s", pattern)
			}
			path := matches[len(matches)-1]
			name := filepath.Base(path)
			destPath := filepath.Join("/tmp", name)
			mods = append(mods,
				ansible.Module{
					Name: "copy",
					Args: []string{
						"src=" + path,
						"dest=" + destPath,
					},
				},
				ansible.Module{
					Name: "yum",
					Args: []string{
						"name=" + destPath,
						"state=installed",
						"update_cache=yes",
						// disablerepo
						// enablerepo
					},
				},
				ansible.Module{
					Name: "file",
					Args: []string{
						"name=" + destPath,
						"state=absent",
					},
				},
			)
		}
		pb.Modules = append(pb.Modules, mods...)
	}

	pb.Modules = append(pb.Modules,
		ansible.Module{
			Name: "copy",
			Args: []string{
				"remote_src=yes",
				"src=/opt/yunion/share/lbagent/yunion-lbagent.service",
				"dest=/etc/systemd/system/yunion-lbagent.service",
			},
		},
		ansible.Module{
			Name: "systemd",
			Args: []string{
				"name=yunion-lbagent",
				"enabled=yes",
				"state=restarted",
				"daemon_reload=yes",
			},
		},
	)
	return pb, nil
}

func (lbagent *SLoadbalancerAgent) undeploy(ctx context.Context, userCred mcclient.TokenCredential, host ansible.Host) (*ansible.Playbook, error) {
	pb := &ansible.Playbook{
		Inventory: ansible.Inventory{
			Hosts: []ansible.Host{host},
		},
		Modules: []ansible.Module{
			{
				Name: "systemd",
				Args: []string{
					"name=yunion-lbagent",
					"enabled=no",
					"state=stopped",
					"daemon_reload=yes",
				},
			},
			{
				Name: "shell",
				Args: []string{
					`pkill keepalived; pkill telegraf; pkill gobetween; pkill haproxy; true`,
				},
			},
			{
				Name: "package",
				Args: []string{
					"name=yunion-lbagent",
					"state=absent",
				},
			},
		},
	}
	// we leave alone
	//
	//  - /etc/yum.repos.d/yunion.repo
	//  - /etc/yunion/lbagent.conf
	//  - state of packages keepalived, haproxy, gobetween, telegraf
	//
	// This decision is unlikely to cause harm.  These content are likely still needed by users
	return pb, nil
}

func (lbagent *SLoadbalancerAgent) validateHost(ctx context.Context, userCred mcclient.TokenCredential, host *ansible.Host) error {
	name := strings.TrimSpace(host.Name)
	if len(name) == 0 {
		return httperrors.NewBadRequestError("empty host name")
	}
	switch {
	case regutils.MatchIP4Addr(name):
	case strings.HasPrefix(name, "host:"):
		name = strings.TrimSpace(name[len("host:"):])
		obj, err := db.FetchByIdOrName(HostManager, userCred, name)
		if err != nil {
			return httperrors.NewNotFoundError("find host %s: %v", name, err)
		}
		host := obj.(*SHost)
		if host.IsManaged() {
			return httperrors.NewBadRequestError("lbagent cannot be deployed on managed host")
		}
	case strings.HasPrefix(name, "server:"):
		name = name[len("server:"):]
		fallthrough
	default:
		obj, err := db.FetchByIdOrName(GuestManager, userCred, name)
		if err != nil {
			return httperrors.NewNotFoundError("find guest %s: %v", name, err)
		}
		guest := obj.(*SGuest)
		if utils.IsInStringArray(guest.Hypervisor, compute_apis.PUBLIC_CLOUD_HYPERVISORS) {
			return httperrors.NewBadRequestError("lbagent cannot be deployed on public guests")
		}
	}
	return nil
}

func (lbagent *SLoadbalancerAgent) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := &compute_apis.LoadbalancerAgentDeployInput{}
	if err := data.Unmarshal(input); err != nil {
		return nil, httperrors.NewBadRequestError("unmarshal input", err)
	}
	host := input.Host
	for _, k := range []string{"user", "pass", "proj"} {
		if v, ok := host.GetVar(k); !ok {
			return nil, httperrors.NewBadRequestError("host missing %s field", k)
		} else if v == "" {
			return nil, httperrors.NewBadRequestError("empty host %s field", k)
		}
	}
	{
		cli := mcclient.NewClient(options.Options.AuthURL, 10, false, true, "", "")
		token, err := cli.Authenticate(host.Vars["user"], host.Vars["pass"], "", host.Vars["proj"], "")
		if err != nil {
			return nil, httperrors.NewBadRequestError("authenticate error: %v", err)
		}
		if !token.HasSystemAdminPrivilege() {
			return nil, httperrors.NewBadRequestError("user must have system admin privileges")
		}
	}
	if err := lbagent.validateHost(ctx, userCred, &host); err != nil {
		return nil, err
	}
	host.SetVar("region", options.Options.Region)
	host.SetVar("auth_uri", options.Options.AuthURL)
	host.SetVar("id", lbagent.Id)
	host.SetVar("ansible_become", "yes")

	pb, err := lbagent.deploy(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	pbId := ""
	if lbagent.Deployment != nil && lbagent.Deployment.AnsiblePlaybook != "" {
		pbId = lbagent.Deployment.AnsiblePlaybook
	}
	pbModel, err := lbagent.updateOrCreatePbModel(ctx, userCred, pbId, lbagent.Name+"-"+lbagent.Id[:7], pb)
	if err != nil {
		return nil, err
	}
	logclient.AddActionLogWithContext(ctx, lbagent, "提交部署任务", pbModel, userCred, true)

	if _, err := db.Update(lbagent, func() error {
		lbagent.Deployment = &SLoadbalancerAgentDeployment{
			Host:            input.Host.Name,
			AnsiblePlaybook: pbModel.Id,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, err
}

func (lbagent *SLoadbalancerAgent) updateOrCreatePbModel(ctx context.Context,
	userCred mcclient.TokenCredential,
	pbId string,
	pbName string,
	pb *ansible.Playbook,
) (*mcclient_models.AnsiblePlaybook, error) {
	cliSess := auth.GetSession(ctx, userCred, "", "")

	if pbId == "" {
		pbJson, err := mcclient_modules.AnsiblePlaybooks.Get(cliSess, pbName, nil)
		if err == nil {
			pbModel := &mcclient_models.AnsiblePlaybook{}
			if err := pbJson.Unmarshal(pbModel); err == nil {
				pbId = pbModel.Id
			}
		}
	}

	var pbJson jsonutils.JSONObject
	if pbId != "" {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookUpdateInput{
			Name:     pbName,
			Playbook: *pb,
		}
		params := ansiblePbInput.JSON(ansiblePbInput)
		pbJson, err = mcclient_modules.AnsiblePlaybooks.Update(cliSess, pbId, params)
		if err != nil {
			return nil, errors.WithMessage(err, "update ansibleplaybook")
		}
	} else {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookCreateInput{
			Name:     pbName,
			Playbook: *pb,
		}
		params := ansiblePbInput.JSON(ansiblePbInput)
		pbJson, err = mcclient_modules.AnsiblePlaybooks.Create(cliSess, params)
		if err != nil {
			return nil, errors.WithMessage(err, "create ansibleplaybook")
		}
	}

	pbModel := &mcclient_models.AnsiblePlaybook{}
	if err := pbJson.Unmarshal(pbModel); err != nil {
		return nil, errors.WithMessage(err, "unmarshal ansibleplaybook")
	}
	return pbModel, nil
}

func (lbagent *SLoadbalancerAgent) PerformUndeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	deployment := lbagent.Deployment
	if deployment == nil || deployment.Host == "" {
		return nil, httperrors.NewConflictError("No previous deployment info available")
	}
	host := ansible.Host{
		Name: deployment.Host,
		Vars: map[string]string{
			"ansible_become": "yes",
		},
	}
	pb, err := lbagent.undeploy(ctx, userCred, host)
	if err != nil {
		return nil, err
	}
	pbModel, err := lbagent.updateOrCreatePbModel(ctx, userCred,
		deployment.AnsiblePlaybookUndeployment,
		lbagent.Name+"-"+lbagent.Id[:7]+"-undeploy",
		pb)
	if err != nil {
		return nil, err
	}
	logclient.AddActionLogWithContext(ctx, lbagent, "提交下线任务", pbModel, userCred, true)
	if _, err := db.Update(lbagent, func() error {
		lbagent.Deployment.AnsiblePlaybookUndeployment = pbModel.Id
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, nil
}

const (
	lbagentConfTmpl = `
region = '{{ region }}'
auth_uri = '{{ auth_uri }}'
admin_user = '{{ user }}'
admin_password = '{{ pass }}'
admin_tenant_name = '{{ proj }}'

data_preserve_n = 10
base_data_dir = "/opt/cloud/workspace/lbagent"

api_lbagent_id = '{{ id }}'
api_lbagent_hb_interval = 60

api_sync_interval = 5
api_list_batch_size = 2048
`

	yunionRepoTmpl = `
[yunion]
name=Packages for Yunion- $basearch
baseurl={{ repo_base_url }}/updates
failovermethod=priority
enabled=1
gpgcheck=0
sslverify={{ repo_sslverify }}

[yunion-extra]
name=Extra Packages for Yunion - $basearch
baseurl={{ repo_base_url }}/packages
failovermethod=priority
enabled=1
gpgcheck=0
sslverify={{ repo_sslverify }}
`
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAgentDeployment{}), func() gotypes.ISerializable {
		return &SLoadbalancerAgentDeployment{}
	})
}
