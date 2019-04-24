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

func (o SecretListOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceResourceListOptions.Params()
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params
}

type RegistrySecretCreateOptions struct {
	NamespaceWithClusterOptions
	NAME     string `help:"Name of secret"`
	Server   string `help:"Docker registry server, e.g. 'https://index.docker.io/v1/'" required:"true"`
	User     string `help:"Docker registry user" required:"true"`
	Password string `help:"Docker registry password" required:"true"`
	Email    string `help:"Docker registry user email"`
}

func (o RegistrySecretCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.Server), "server")
	params.Add(jsonutils.NewString(o.User), "user")
	params.Add(jsonutils.NewString(o.Password), "password")
	if o.Email != "" {
		params.Add(jsonutils.NewString(o.Email), "email")
	}
	return params, nil
}
