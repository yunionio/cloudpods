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

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_baseOptions(t *testing.T) {
	opt := newBaseOptions_x86_64()
	assert := assert.New(t)

	// test object
	assert.Equal("-object iothread,id=iothread0", opt.Object("iothread", map[string]string{"id": "iothread0"}))
	// test chardev
	assert.Equal("-chardev socket,id=test", opt.Chardev("socket", "test", ""))
	assert.Equal("-chardev socket,id=test,name=tname", opt.Chardev("socket", "test", "tname"))
	assert.Equal("-chardev socket,id=test,port=1234,host=127.0.0.1,nodelay,server,nowait", opt.MonitorChardev("test", 1234, "127.0.0.1"))
	assert.Equal([]string{
		"-chardev socket,id=testdev,port=1234,host=127.0.0.1,nodelay,server,nowait",
		"-mon chardev=testdev,id=test,mode=readline",
	}, getMonitorOptions(opt, &Monitor{
		Id:   "test",
		Port: 1234,
		Mode: "readline",
	}))
	// test device
	assert.Equal("-device isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc", opt.Device("isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc"))
	// test vnc
	assert.Equal("-vnc :5900,password", opt.VNC(5900, true))
	assert.Equal("-vnc :5900", opt.VNC(5900, false))
}
