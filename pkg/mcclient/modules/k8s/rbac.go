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
