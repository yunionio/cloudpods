package monitor

import (
	"testing"
	"time"

	"yunion.io/x/log"
)

func TestHmpMonitor_Connect(t *testing.T) {
	onConnected := func() { log.Infof("Monitor Connected") }
	onDisConnect := func(error) { log.Infof("Monitor DisConnect") }
	onTimeout := func(error) { log.Infof("Monitor Timeout") }
	m := NewHmpMonitor(onDisConnect, onTimeout, onConnected)
	var host = "127.0.0.1"
	var port = 55901
	m.Connect(host, port)
	rawCallBack := func(res string) { log.Infof("OnCallback: %s", res) }
	m.Query("info block", rawCallBack)
	m.Query("unknown cmd", rawCallBack)

	statusCallBack := func(res string) { log.Infof("OnStatusCallback %s", res) }
	m.QueryStatus(statusCallBack)
	m.Disconnect()
	time.Sleep(3 * time.Second)
}
