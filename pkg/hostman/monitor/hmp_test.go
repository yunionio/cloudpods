// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"testing"
	"time"
)

func TestHmpMonitor_Connect(t *testing.T) {
	onConnected := func() { t.Logf("Monitor Connected") }
	onDisConnect := func(error) { t.Logf("Monitor DisConnect") }
	onTimeout := func(error) { t.Logf("Monitor Timeout") }
	m := NewHmpMonitor("fake_server", onDisConnect, onTimeout, onConnected)
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
