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
	RbacRoles        *RbacRoleManager
	RbacRoleBindings *RbacRoleBindingManager
	ServiceAccounts  *ServiceAccountManager
)

type RbacRoleManager struct {
	*NamespaceResourceManager
}

type RbacRoleBindingManager struct {
	*NamespaceResourceManager
}

type ServiceAccountManager struct {
	*NamespaceResourceManager
}

func init() {
	RbacRoles = &RbacRoleManager{
		NewNamespaceResourceManager("rbacrole", "rbacroles", NewNamespaceCols(), NewColumns("Type"))}

	RbacRoleBindings = &RbacRoleBindingManager{
		NewNamespaceResourceManager("rbacrolebinding", "rbacrolebindings", NewNamespaceCols(), NewColumns("Type"))}

	ServiceAccounts = &ServiceAccountManager{
		NewNamespaceResourceManager("serviceaccount", "serviceaccounts", NewNamespaceCols(), NewColumns())}

	modules.Register(RbacRoles)
	modules.Register(RbacRoleBindings)
}

func (m RbacRoleManager) GetType(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}

func (m RbacRoleBindingManager) GetType(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}
