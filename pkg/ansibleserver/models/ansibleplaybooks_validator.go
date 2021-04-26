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
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	mcclient "yunion.io/x/onecloud/pkg/mcclient"
	mcclient_auth "yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_models "yunion.io/x/onecloud/pkg/mcclient/models"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
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
		defaultUsername := ansible.PUBLIC_CLOUD_ANSIBLE_USER
		switch {
		case regutils.MatchIP4Addr(name):
		case strings.HasPrefix(name, "host:"):
			var err error
			name = strings.TrimSpace(name[len("host:"):])
			name, err = v.getHostAccessIp(name)
			if err != nil {
				return err
			}
			defaultUsername = "root"
		case strings.HasPrefix(name, "server:"):
			name = name[len("server:"):]
			fallthrough
		default:
			var err error
			name = strings.TrimSpace(name)
			name, err = v.getServerIp(name)
			if err != nil {
				return err
			}
		}
		hosts[i].Name = name
		if username, _ := hosts[i].GetVar("ansible_user"); username == "" {
			if defaultUsername != "" {
				hosts[i].SetVar("ansible_user", defaultUsername)
			}
		}
	}
	// add LF for privateKey
	if len(pb.PrivateKey) > 0 && pb.PrivateKey[len(pb.PrivateKey)-1] != 10 {
		pb.PrivateKey = append(pb.PrivateKey, 10)
	}
	pbJson := jsonutils.Marshal(pb)
	if serialized := pbJson.String(); len(serialized) > PlaybookMaxBytes {
		return httperrors.NewBadRequestError("playbook too big, got %d bytes, exceeding %d",
			len(serialized), PlaybookMaxBytes)
	}
	v.Playbook = pb
	data.Set("playbook", pbJson)
	return nil
}

func (v *ValidatorAnsiblePlaybook) getHostAccessIp(name string) (string, error) {
	sess := mcclient_auth.GetSession(context.Background(), v.userCred, "", "")
	hostJson, err := mcclient_modules.Hosts.Get(sess, name, nil)
	if err != nil {
		return "", httperrors.NewBadRequestError("cannot find host %s", name)
	}
	host := &mcclient_models.Host{}
	if err := hostJson.Unmarshal(host); err != nil {
		return "", httperrors.NewBadRequestError("unmarshal host %s: %v", name, err)
	}
	if host.AccessIp == "" {
		return "", httperrors.NewBadRequestError("host %s has no access ip", name)
	}
	return host.AccessIp, nil
}

func (v *ValidatorAnsiblePlaybook) getServerIp(name string) (string, error) {
	sess := mcclient_auth.GetSession(context.Background(), v.userCred, "", "")
	serverJson, err := mcclient_modules.Servers.Get(sess, name, nil)
	if err != nil {
		return "", httperrors.NewBadRequestError("find server %s: %v", name, err)
	}
	server := &mcclient_models.Server{}
	if err := serverJson.Unmarshal(server); err != nil {
		return "", httperrors.NewBadRequestError("unmarshal server %s: %v", name, err)
	}
	serverNetworks, err := mcclient_models.ParseServerNetworkDetailedString(server.Networks)
	if err != nil {
		return "", httperrors.NewConflictError("parse networks of %s: %v", name, err)
	}
	ips := serverNetworks.GetPrivateIPs()
	if len(ips) == 0 {
		return "", httperrors.NewBadRequestError("server %s has no private ips", name)
	}
	name = ips[0].String()
	return name, nil
}
