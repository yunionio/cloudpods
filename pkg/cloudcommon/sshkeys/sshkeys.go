package sshkeys

import "yunion.io/x/jsonutils"

type SSHKeys struct {
	PublicKey        string
	DeletePublicKey  string
	AdminPublicKey   string
	ProjectPublicKey string
}

func GetKeys(data jsonutils.JSONObject) *SSHKeys {
	var ret = new(SSHKeys)
	ret.PublicKey, _ = data.GetString("public_key")
	ret.DeletePublicKey, _ = data.GetString("delete_public_key")
	ret.AdminPublicKey, _ = data.GetString("admin_public_key")
	ret.ProjectPublicKey, _ = data.GetString("project_public_key")
	return ret
}
