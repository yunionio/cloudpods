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

package collectors

import (
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	result, err := Ping([]string{
		"114.114.114.114",
		"118.187.65.237",
		"10.168.26.254",
		"10.168.26.26",
		"192.30.253.113",
	}, 5, time.Second, true)
	if err != nil {
		// ignore error
		t.Logf("ping error %s", err)
	}
	for k, v := range result {
		t.Logf("%s: %s", k, v.String())
	}
}
