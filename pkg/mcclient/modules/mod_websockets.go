package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SWebsocketManager struct {
	ResourceManager
}

var (
	Websockets SWebsocketManager
)

func (this *SWebsocketManager) DoNotify(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	this.PerformAction(s, "", "notify", params)
	return ret, nil
}

func init() {

	Websockets = SWebsocketManager{NewWebsocketManager("websocket", "websockets",
		// user view
		[]string{},
		[]string{}, // admin view
	)}

	register(&Websockets)
}
