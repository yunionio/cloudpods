package socket

import "yunion.io/x/onecloud/pkg/appsrv"

var socketWorker *appsrv.SWorkerManager

func GetSocketWorker() *appsrv.SWorkerManager {
	if socketWorker == nil {
		// allow 1024 user online
		socketWorker = appsrv.NewWorkerManager("ws", 1024, 1, false)
	}

	return socketWorker
}
