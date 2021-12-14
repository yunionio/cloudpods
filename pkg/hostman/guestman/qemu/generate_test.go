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

	"yunion.io/x/log"
)

func TestGenerateStartCommand(t *testing.T) {
	assert := assert.New(t)
	log.Errorf("---%v", assert)
	input := &GenerateStartOptionsInput{
		QemuVersion:          Version_2_12_1,
		QemuArch:             Arch_x86_64,
		UUID:                 "uuid-xxxx-xxxx",
		Mem:                  1024,
		Cpu:                  2,
		Name:                 "test-vm",
		OsName:               OS_NAME_LINUX,
		OVNIntegrationBridge: "brvpc",
		HomeDir:              "/opt/cloud/workspace/servers/sid",
		HugepagesEnabled:     true,
		PidFilePath:          "/opt/cloud/workspace/servers/sid/pid",
	}
	cmd, err := GenerateStartOptions(input)
	log.Errorf("cmd: %s", cmd)
	log.Errorf("error: %s", err)
}
