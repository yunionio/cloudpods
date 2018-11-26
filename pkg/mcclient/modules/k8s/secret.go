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
