package guestfs

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
)

type SDeployInfo struct {
	publicKey *sshkeys.SSHKeys
	deploys   jsonutils.JSONObject
	password  string
	isInit    bool
}
