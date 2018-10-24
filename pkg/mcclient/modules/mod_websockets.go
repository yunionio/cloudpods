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

func init() {

	Websockets = SWebsocketManager{NewWebsocketManager("websocket", "websockets",
		// user view
		[]string{},
		[]string{}, // admin view
	)}

	register(&Websockets)
}
