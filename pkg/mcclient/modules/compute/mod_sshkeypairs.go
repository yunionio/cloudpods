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

func (this *SSshkeypairManager) FetchProjectPrivateKeys(ctx context.Context, userCred mcclient.TokenCredential) ([]string, error) {
	s := auth.GetSession(ctx, userCred, "")
	return this.FetchProjectPrivateKeysBySession(ctx, s)
}

func (this *SSshkeypairManager) FetchProjectPrivateKeysBySession(ctx context.Context, s *mcclient.ClientSession) ([]string, error) {
	kp, err := this.FetchProjectKeypairsBySession(ctx, s)
	if err != nil {
		return nil, errors.Wrap(err, "FetchProjectKeypairsBySession")
	}
	keys := make([]string, 0, len(kp))
	for _, kp := range kp {
		keys = append(keys, kp.PrivateKey)
	}
	return keys, nil
}

func (this *SSshkeypairManager) FetchAdminPrivateKeysBySession(ctx context.Context, s *mcclient.ClientSession) ([]string, error) {
	kp, err := this.FetchAdminKeypairsBySession(ctx, s)
	if err != nil {
		return nil, errors.Wrap(err, "FetchAdminKeypairsBySession")
	}
	keys := make([]string, 0, len(kp))
	for _, kp := range kp {
		keys = append(keys, kp.PrivateKey)
	}
	return keys, nil
}

func (this *SSshkeypairManager) FetchAdminKeypairsBySession(ctx context.Context, s *mcclient.ClientSession) ([]models.SshKeypair, error) {
	return this.fetchKeypairsBySession(ctx, s, true)
}

func (this *SSshkeypairManager) FetchProjectKeypairsBySession(ctx context.Context, s *mcclient.ClientSession) ([]models.SshKeypair, error) {
	return this.fetchKeypairsBySession(ctx, s, false)
}

func (this *SSshkeypairManager) fetchKeypairsBySession(ctx context.Context, s *mcclient.ClientSession, isAdmin bool) ([]models.SshKeypair, error) {
	jd := jsonutils.NewDict()
	if isAdmin {
		jd.Set("admin", jsonutils.JSONTrue)
	}
	r, err := this.List(s, jd)
	if err != nil {
		return nil, errors.Wrap(err, "get admin ssh key")
	}
	if len(r.Data) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no ssh key found")
	}
	return unmarshalSshKeypairs(r.Data[0])
}

func (this *SSshkeypairManager) FetchKeypairsByProject(ctx context.Context, s *mcclient.ClientSession, projectId string) ([]models.SshKeypair, error) {
	jd := jsonutils.NewDict()
	if len(projectId) == 0 {
		projectId = s.GetProjectId()
	}
	r, err := Sshkeypairs.GetById(s, projectId, jd)
	if err != nil {
		return nil, errors.Wrap(err, "get project ssh key")
	}
	return unmarshalSshKeypairs(r)
}

func unmarshalSshKeypairs(jr jsonutils.JSONObject) ([]models.SshKeypair, error) {
	kps := make([]models.SshKeypair, 0, 2)
	if jr.Contains("keypairs") {
		if err := jr.Unmarshal(&kps, "keypairs"); err != nil {
			return nil, errors.Wrap(err, "unmarshal ssh key")
		}
	} else {
		kp := models.SshKeypair{}
		if err := jr.Unmarshal(&kp); err != nil {
			return nil, errors.Wrap(err, "unmarshal ssh key")
		}
		kps = append(kps, kp)
	}
	return kps, nil
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
