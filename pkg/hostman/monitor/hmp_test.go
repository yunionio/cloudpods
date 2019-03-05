package monitor

import (
	"testing"
	"time"
)

func TestHmpMonitor_Connect(t *testing.T) {
	onConnected := func() { t.Logf("Monitor Connected") }
	onDisConnect := func(error) { t.Logf("Monitor DisConnect") }
	onTimeout := func(error) { t.Logf("Monitor Timeout") }
	m := NewHmpMonitor(onDisConnect, onTimeout, onConnected)
	var host = "127.0.0.1"
	var port = 55901
	m.Connect(host, port)
	rawCallBack := func(res string) { t.Logf("OnCallback: %s", res) }
	m.Query("info block", rawCallBack)
	m.Query("unknown cmd", rawCallBack)

	statusCallBack := func(res string) { t.Logf("OnStatusCallback %s", res) }
	m.QueryStatus(statusCallBack)

	m.GetBlockJobCounts(func(res int) { t.Logf("GetBlockJobCounts %d", res) })
	m.GetCpuCount(func(res int) { t.Logf("GetCpuCount %d", res) })
	time.Sleep(3 * time.Second)
	m.Disconnect()
}
