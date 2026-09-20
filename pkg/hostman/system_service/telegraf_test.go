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

package system_service

import (
	"strings"
	"testing"
)

func TestGetConfigXPUSMI(t *testing.T) {
	s := NewTelegrafService()

	conf := s.GetConfig(map[string]interface{}{
		TELEGRAF_INPUT_XPUSMI: map[string]interface{}{
			TELEGRAF_INPUT_CONF_BIN_PATH: "/usr/local/bin/xpu-smi",
			TELEGRAF_INPUT_CONF_LIB_PATH: "/usr/local/xpu/so",
		},
	})
	for _, want := range []string{
		"[[inputs.xpusmi]]",
		`  bin_path = "/usr/local/bin/xpu-smi"`,
		`  lib_path = "/usr/local/xpu/so"`,
	} {
		if !strings.Contains(conf, want) {
			t.Errorf("telegraf config does not contain %q:\n%s", want, conf)
		}
	}
}

func TestGetConfigXPUSMIOmitsEmptyLibPath(t *testing.T) {
	s := NewTelegrafService()

	conf := s.GetConfig(map[string]interface{}{
		TELEGRAF_INPUT_XPUSMI: map[string]interface{}{
			TELEGRAF_INPUT_CONF_BIN_PATH: "/opt/xpu/bin/xpu-smi",
			TELEGRAF_INPUT_CONF_LIB_PATH: "",
		},
	})
	if !strings.Contains(conf, "[[inputs.xpusmi]]") {
		t.Errorf("telegraf config does not contain [[inputs.xpusmi]]:\n%s", conf)
	}
	if !strings.Contains(conf, `  bin_path = "/opt/xpu/bin/xpu-smi"`) {
		t.Errorf("telegraf config does not contain the expected bin_path:\n%s", conf)
	}
	if strings.Contains(conf, "lib_path") {
		t.Errorf("telegraf config should not contain lib_path:\n%s", conf)
	}
}

func TestGetConfigWithoutXPUSMI(t *testing.T) {
	s := NewTelegrafService()

	conf := s.GetConfig(map[string]interface{}{})
	if strings.Contains(conf, "[[inputs.xpusmi]]") {
		t.Errorf("telegraf config should not contain [[inputs.xpusmi]]:\n%s", conf)
	}
}
