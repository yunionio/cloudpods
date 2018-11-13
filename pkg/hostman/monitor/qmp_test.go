package monitor

import (
	"testing"
	"time"

	"yunion.io/x/log"
)

func TestQmpMonitor_Connect(t *testing.T) {
	onConnected := func() { log.Infof("Monitor Connected") }
	onDisConnect := func(error) { log.Infof("Monitor DisConnect") }
	onTimeout := func(error) { log.Infof("Monitor Timeout") }
	m := NewQmpMonitor(onConnected, onDisConnect, onTimeout)
	var host = "127.0.0.1"
	var port = 56101
	m.Connect(host, port)
	rawCallBack := func(res *Response) { log.Infof("OnCallback %s", res) }
	// cmd0 := &Command{Execute: "qmp_capabilities"}
	// m.Query(cmd0, rawCallBack)
	cmd1 := &Command{
		Execute: "human-monitor-command",
		Args:    map[string]string{"command-line": "info block"},
	}
	m.Query(cmd1, rawCallBack)

	statusCallBack := func(res string) { log.Infof("OnStatusCallback %s", res) }
	m.QueryStatus(statusCallBack)
	time.Sleep(3 * time.Second)
}
