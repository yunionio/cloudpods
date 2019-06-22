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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

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
)

type SLoadbalancerAgentDeployment struct {
	Host            string
	AnsiblePlaybook string
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
					"validate_certs=no",
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
				"state=started",
				"daemon_reload=yes",
			},
		},
	)
	return pb, nil
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
	host.SetVar("region", options.Options.Region)
	host.SetVar("auth_uri", options.Options.AuthURL)
	host.SetVar("id", lbagent.Id)
	host.SetVar("ansible_become", "yes")

	pb, err := lbagent.deploy(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	cliSess := auth.GetSession(ctx, userCred, "", "")
	var pbJson jsonutils.JSONObject
	if lbagent.Deployment != nil && lbagent.Deployment.AnsiblePlaybook != "" {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookUpdateInput{
			Name:     lbagent.Name + "-" + lbagent.Id[:7],
			Playbook: *pb,
		}
		params := ansiblePbInput.JSON(ansiblePbInput)
		pbJson, err = mcclient_modules.AnsiblePlaybooks.Update(cliSess, lbagent.Deployment.AnsiblePlaybook, params)
		if err != nil {
			return nil, errors.WithMessage(err, "update ansibleplaybook")
		}
	} else {
		var err error
		ansiblePbInput := &ansible_apis.AnsiblePlaybookCreateInput{
			Name:     lbagent.Name + "-" + lbagent.Id[:7],
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

[yunion-extra]
name=Extra Packages for Yunion - $basearch
baseurl={{ repo_base_url }}/packages
failovermethod=priority
enabled=1
gpgcheck=0
`
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAgentDeployment{}), func() gotypes.ISerializable {
		return &SLoadbalancerAgentDeployment{}
	})
}
