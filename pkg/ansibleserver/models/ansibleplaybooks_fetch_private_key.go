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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mcclient_models "yunion.io/x/onecloud/pkg/mcclient/models"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
)

func fetchPrivateKey(ctx context.Context, userCred mcclient.TokenCredential) (string, error) {
	s := auth.GetSession(ctx, userCred, "", "")
	jd := jsonutils.NewDict()
	var jr jsonutils.JSONObject
	if userCred.HasSystemAdminPrivilege() {
		jd.Set("admin", jsonutils.JSONTrue)
		r, err := mcclient_modules.Sshkeypairs.List(s, jd)
		if err != nil {
			return "", errors.WithMessage(err, "get admin ssh key")
		}
		jr = r.Data[0]
	} else {
		r, err := mcclient_modules.Sshkeypairs.GetById(s, userCred.GetProjectId(), jd)
		if err != nil {
			return "", errors.WithMessage(err, "get project ssh key")
		}
		jr = r
	}
	kp := &mcclient_models.SshKeypair{}
	if err := jr.Unmarshal(kp); err != nil {
		return "", errors.WithMessage(err, "unmarshal ssh key")
	}
	if kp.PrivateKey == "" {
		return "", errors.New("empty ssh key")
	}
	return kp.PrivateKey, nil
}
