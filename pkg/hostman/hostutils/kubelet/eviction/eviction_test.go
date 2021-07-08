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

package eviction

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	type tCase struct {
		name       string
		input      string
		assertFunc func(*testing.T, Config) error
	}

	cases := []tCase{
		{
			name:  "Should has default config with empty input",
			input: ``,
			assertFunc: func(t *testing.T, config Config) error {
				memBytes, _ := config.GetHard().GetMemoryAvailable().Value.Quantity.AsInt64()
				assert.Equal(t, int64(1024*1024*100), memBytes)
				assert.Equal(t, float32(0.1), config.GetHard().GetNodeFsAvailable().Value.Percentage)
				assert.Equal(t, float32(0.05), config.GetHard().GetNodeFsInodesFree().Value.Percentage)
				assert.Equal(t, float32(0.15), config.GetHard().GetImageFsAvailable().Value.Percentage)
				return nil
			},
		},
		{
			name: "With all config",
			input: `
evictionHard:
  imagefs.available: 25%
  memory.available: 1024Mi
  nodefs.available: 15%
  nodefs.inodesFree: 10%`,
			assertFunc: func(t *testing.T, config Config) error {
				memBytes, _ := config.GetHard().GetMemoryAvailable().Value.Quantity.AsInt64()
				assert.Equal(t, int64(1024*1024*1024), memBytes)
				assert.Equal(t, float32(0.25), config.GetHard().GetImageFsAvailable().Value.Percentage)
				assert.Equal(t, float32(0.15), config.GetHard().GetNodeFsAvailable().Value.Percentage)
				assert.Equal(t, float32(0.1), config.GetHard().GetNodeFsInodesFree().Value.Percentage)
				return nil
			},
		},
		{
			name: "Mixed config with default",
			input: `
evictionHard:
  imagefs.available: 5%
  nodefs.inodesFree: 10%`,
			assertFunc: func(t *testing.T, config Config) error {
				memBytes, _ := config.GetHard().GetMemoryAvailable().Value.Quantity.AsInt64()
				assert.Equal(t, int64(1024*1024*100), memBytes)
				assert.Equal(t, float32(0.05), config.GetHard().GetImageFsAvailable().Value.Percentage)
				assert.Equal(t, float32(0.1), config.GetHard().GetNodeFsAvailable().Value.Percentage)
				assert.Equal(t, float32(0.1), config.GetHard().GetNodeFsInodesFree().Value.Percentage)
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(st *testing.T) {
			if config, err := NewConfig([]byte(tc.input)); err != nil {
				st.Errorf("[%s] NewConfig error: %v", tc.name, err)
			} else {
				str, _ := json.MarshalIndent(config, "", "  ")
				st.Logf("%s", str)
				if err := tc.assertFunc(st, config); err != nil {
					st.Errorf("[%s] assert error: %v", tc.name, err)
				}
			}
		})
	}
}
