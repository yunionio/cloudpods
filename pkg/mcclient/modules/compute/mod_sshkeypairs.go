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

package compute

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SSshkeypairManager struct {
	modulebase.ResourceManager
}

func (this *SSshkeypairManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	url := "/sshkeypairs"
	if params != nil {
		if queryStr := params.QueryString(); queryStr != "" {
			url = fmt.Sprintf("%s?%s", url, queryStr)
		}
	}
	body, err := modulebase.Get(this.ResourceManager, s, url, "sshkeypair")
	if err != nil {
		return nil, err
	}
	result := printutils.ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

func (this *SSshkeypairManager) FetchPrivateKey(ctx context.Context, userCred mcclient.TokenCredential) (string, error) {
	s := auth.GetSession(ctx, userCred, "")
	return this.FetchPrivateKeyBySession(ctx, s)
}

func (this *SSshkeypairManager) FetchPrivateKeyBySession(ctx context.Context, s *mcclient.ClientSession) (string, error) {
	kp, err := this.FetchKeypairBySession(ctx, s)
	if err != nil {
		return "", errors.Wrap(err, "FetchKeypairBySession")
	}
	return kp.PrivateKey, nil
}

func (this *SSshkeypairManager) FetchKeypairBySession(ctx context.Context, s *mcclient.ClientSession) (*models.SshKeypair, error) {
	userCred := s.GetToken()
	jd := jsonutils.NewDict()
	var jr jsonutils.JSONObject
	if userCred.HasSystemAdminPrivilege() {
		jd.Set("admin", jsonutils.JSONTrue)
		r, err := Sshkeypairs.List(s, jd)
		if err != nil {
			return nil, errors.Wrap(err, "get admin ssh key")
		}
		jr = r.Data[0]
	} else {
		r, err := Sshkeypairs.GetById(s, userCred.GetProjectId(), jd)
		if err != nil {
			return nil, errors.Wrap(err, "get project ssh key")
		}
		jr = r
	}
	kp := &models.SshKeypair{}
	if err := jr.Unmarshal(kp); err != nil {
		return nil, errors.Wrap(err, "unmarshal ssh key")
	}
	return kp, nil
}

var (
	Sshkeypairs SSshkeypairManager
)

func init() {
	Sshkeypairs = SSshkeypairManager{modules.NewComputeManager("sshkeypair", "sshkeypairs",
		[]string{},
		[]string{})}

	modules.RegisterCompute(&Sshkeypairs)
}
