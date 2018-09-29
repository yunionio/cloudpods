package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUpdateManager struct {
	ResourceManager
}

var (
	Updates SUpdateManager
)

func (this *SUpdateManager) DoUpdate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	this.PerformAction(s, "", "", params)
	return ret, nil
}

func init() {

	Updates = SUpdateManager{NewAutoUpdateManager("update", "updates",
		// user view
		[]string{"localVersion", "remoteVersion", "status", "updateAvailable"},
		[]string{}, // admin view
	)}

	register(&Updates)
}
