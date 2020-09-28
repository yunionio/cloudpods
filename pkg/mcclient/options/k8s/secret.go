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

package k8s

import (
	"yunion.io/x/jsonutils"
)

type SecretListOptions struct {
	NamespaceResourceListOptions
	Type string `help:"Secret type"`
}

func (o SecretListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.NamespaceResourceListOptions.Params()
	if err != nil {
		return nil, err
	}
	if o.Type != "" {
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString(o.Type), "type")
	}
	return params, nil
}

type RegistrySecretCreateOptions struct {
	SecretCreateOptions
	Server   string `help:"Docker registry server, e.g. 'https://index.docker.io/v1/'" required:"true"`
	User     string `help:"Docker registry user" required:"true"`
	Password string `help:"Docker registry password" required:"true"`
	Email    string `help:"Docker registry user email"`
}

func (o RegistrySecretCreateOptions) Params() (jsonutils.JSONObject, error) {
	input, err := o.SecretCreateOptions.Params("kubernetes.io/dockerconfigjson")
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.Server), "server")
	params.Add(jsonutils.NewString(o.User), "user")
	params.Add(jsonutils.NewString(o.Password), "password")
	if o.Email != "" {
		params.Add(jsonutils.NewString(o.Email), "email")
	}
	input.Add(params, "dockerConfigJson")
	return input, nil
}

type SecretCreateOptions struct {
	NamespaceWithClusterOptions
	NAME string `help:"Name of secret"`
}

func (o SecretCreateOptions) Params(typ string) (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(typ), "type")
	return params, nil
}

type CephCSISecretCreateOptions struct {
	SecretCreateOptions
	USERID  string `help:"User id"`
	USERKEY string `help:"User key"`
}

func (o CephCSISecretCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.SecretCreateOptions.Params("yunion.io/ceph-csi")
	if err != nil {
		return nil, err
	}
	conf := jsonutils.NewDict()
	conf.Add(jsonutils.NewString(o.USERID), "userId")
	conf.Add(jsonutils.NewString(o.USERKEY), "userKey")
	params.Add(conf, "cephCSI")
	return params, nil
}
