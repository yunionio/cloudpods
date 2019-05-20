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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type ValidatorAnsiblePlaybook struct {
	validators.Validator
	Playbook *ansible.Playbook

	userCred mcclient.TokenCredential
}

func NewAnsiblePlaybookValidator(key string, userCred mcclient.TokenCredential) *ValidatorAnsiblePlaybook {
	v := &ValidatorAnsiblePlaybook{
		Validator: validators.Validator{Key: key},
		userCred:  userCred,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorAnsiblePlaybook) Validate(data *jsonutils.JSONDict) error {
	pb := ansible.NewPlaybook()
	err := data.Unmarshal(pb, "playbook")
	if err != nil {
		return httperrors.NewBadRequestError("unmarshaling json: %v", err)
	}
	hosts := pb.Inventory.Hosts
	for i := range hosts {
		name := strings.TrimSpace(hosts[i].Name)
		if len(name) == 0 {
			return httperrors.NewBadRequestError("empty host name")
		}
		switch {
		case regutils.MatchIP4Addr(name):
			continue
		case strings.HasPrefix(name, "host:"):
			name = strings.TrimSpace(name[len("host:"):])
			m, err := db.FetchByIdOrName(HostManager, v.userCred, name)
			if err != nil {
				return httperrors.NewBadRequestError("cannot find host %s", name)
			}
			if m.(*SHost).AccessIp == "" {
				return httperrors.NewBadRequestError("host %s has no access ip", name)
			}
			name = m.(*SHost).AccessIp
		case strings.HasPrefix(name, "server:"):
			name = name[len("server:"):]
			fallthrough
		default:
			name = strings.TrimSpace(name)
			m, err := db.FetchByIdOrName(GuestManager, v.userCred, name)
			if err != nil {
				return httperrors.NewBadRequestError("cannot find guest %s", name)
			}
			ips := m.(*SGuest).GetPrivateIPs()
			if len(ips) == 0 {
				return httperrors.NewBadRequestError("guest %s has no private ips", name)
			}
			name = ips[0]
		}
		hosts[i].Name = name
		if username, _ := hosts[i].GetVar("ansible_user"); username == "" {
			hosts[i].SetVar("ansible_user", ansible.PUBLIC_CLOUD_ANSIBLE_USER)
		}
	}
	v.Playbook = pb
	pbJson := jsonutils.Marshal(pb)
	data.Set("playbook", pbJson)
	return nil
}
