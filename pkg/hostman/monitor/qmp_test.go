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

	"yunion.io/x/log"
)

func TestQmpMonitor_Connect(t *testing.T) {
	onConnected := func() { log.Infof("Monitor Connected") }
	onDisConnect := func(error) { log.Infof("Monitor DisConnect") }
	onTimeout := func(error) { log.Infof("Monitor Timeout") }
	m := NewQmpMonitor("", "", onDisConnect, onTimeout, onConnected, nil)
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
	m.Disconnect()
	time.Sleep(3 * time.Second)
}
