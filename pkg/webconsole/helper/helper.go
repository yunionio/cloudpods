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

package helper

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

func GetValidPrivateKey(host string, port int, username string, projectId string) (string, error) {
	errs := []error{}
	ctx := context.Background()
	admin := auth.GetAdminSession(ctx, o.Options.Region)
	for _, gf := range []func() (jsonutils.JSONObject, error){
		func() (jsonutils.JSONObject, error) {
			if projectId == "" {
				return nil, errors.Error("project_id is empty")
			}
			key, err := compute.Sshkeypairs.GetById(admin, projectId, jsonutils.Marshal(map[string]bool{"admin": true}))
			if err != nil {
				return nil, errors.Wrapf(err, "Sshkeypairs.GetById(%s)", projectId)
			}
			return key, nil
		},
		func() (jsonutils.JSONObject, error) {
			query := jsonutils.NewDict()
			query.Set("admin", jsonutils.JSONTrue)
			ret, err := compute.Sshkeypairs.List(admin, query)
			if err != nil {
				return nil, errors.Wrap(err, "modules.Sshkeypairs.List")
			}
			if len(ret.Data) == 0 {
				return nil, errors.Wrap(httperrors.ErrNotFound, "Not found admin sshkey")
			}
			keys := ret.Data[0]
			return keys, nil
		},
	} {
		key, err := gf()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		privKey, err := key.GetString("private_key")
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "get private_key"))
			continue
		}
		signer, err := ssh.ParsePrivateKey([]byte(privKey))
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ParsePrivateKey"))
			continue
		}
		config := &ssh.ClientConfig{
			Timeout:         time.Second,
			User:            username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
		addr := fmt.Sprintf("%s:%d", host, port)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "dial %s by %s", addr, username))
			continue
		}
		defer client.Close()
		return privKey, nil
	}
	return "", errors.NewAggregate(errs)
}
