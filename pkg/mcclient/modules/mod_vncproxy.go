package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type VNCProxyManager struct {
	ResourceManager
}

func (this *VNCProxyManager) DoConnect(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := "/vncproxy"
	objtype, e := params.GetString("objtype")
	if e == nil && len(objtype) > 0 {
		url = fmt.Sprintf("%s/%s", url, objtype)
	}
	url = fmt.Sprintf("%s/%s", url, id)
	return this._post(s, url, nil, "vncproxy")
}

var (
	VNCProxy VNCProxyManager
)

func init() {
	VNCProxy = VNCProxyManager{NewVNCProxyManager()}

	register(&VNCProxy)
}
