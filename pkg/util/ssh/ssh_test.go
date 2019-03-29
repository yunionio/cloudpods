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

package ssh

import (
	"testing"

	"yunion.io/x/log"
)

const (
	username = "root"
	host     = "10.168.222.245"
	password = "123@openmag"
)

func TestRun(t *testing.T) {
	client, err := NewClient(host, 22, username, password, "")
	if err != nil {
		t.Error(err)
	}
	out, err := client.Run("ls", "uname -a", "date", "hostname")
	if err != nil {
		t.Error(err)
	}
	log.Infof("output: %#v", out)
}
