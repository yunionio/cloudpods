package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Secrets *SecretManager

type SecretManager struct {
	*NamespaceResourceManager
}

func init() {
	Secrets = &SecretManager{
		NewNamespaceResourceManager("secret", "secrets",
			NewNamespaceCols("Type"), NewColumns())}
	modules.Register(Secrets)
}

func (m SecretManager) GetType(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}
