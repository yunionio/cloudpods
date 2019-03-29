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

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Secrets         *SecretManager
	RegistrySecrets *RegistrySecretManager
)

type SecretManager struct {
	*NamespaceResourceManager
}

type RegistrySecretManager struct {
	*SecretManager
}

func init() {
	Secrets = &SecretManager{
		NewNamespaceResourceManager("secret", "secrets",
			NewNamespaceCols("Type"), NewColumns())}

	RegistrySecrets = &RegistrySecretManager{
		SecretManager: &SecretManager{
			NewNamespaceResourceManager("registrysecret", "registrysecrets", NewNamespaceCols(), NewColumns())},
	}

	modules.Register(Secrets)
	modules.Register(RegistrySecrets)
}

func (m SecretManager) GetType(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}
