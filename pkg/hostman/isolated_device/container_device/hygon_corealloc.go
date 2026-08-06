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

package container_device

import (
	"fmt"
	"strconv"
	"strings"
)

// initHygonCoreUsage returns a zero-filled hex string for CU bitmap allocation.
func initHygonCoreUsage(_ int) string {
	return strings.Repeat("0", 16)
}

func addHygonCoreUsage(tot string, c string) (string, error) {
	i := 0
	res := ""
	for {
		if i >= len(tot) || tot[i] == 0 {
			break
		}
		left, _ := strconv.ParseInt(string(tot[i]), 16, 0)
		right, _ := strconv.ParseInt(string(c[i]), 16, 0)
		merged := int(left | right)
		res = fmt.Sprintf("%s%x", res, merged)
		i++
	}
	return res, nil
}

func hygonByteAlloc(b int, req int) (int, int) {
	if req == 0 {
		return 0, 0
	}
	remains := req
	leftstr := fmt.Sprintf("%b", b)
	for len(leftstr) < 4 {
		leftstr = "0" + leftstr
	}
	res := 0
	i := 0
	for i < len(leftstr) {
		res = res * 2
		if leftstr[i] == '0' && remains > 0 {
			remains--
			res = res + 1
		}
		if remains <= 0 {
			break
		}
		i++
	}
	return res, remains
}

func allocHygonCoreUsage(tot string, req int) (string, int, error) {
	i := len(tot) - 1
	res := ""
	remains := req
	for {
		if i < 0 {
			break
		}
		left, _ := strconv.ParseInt(string(tot[i]), 16, 0)
		alloc, newRemains := hygonByteAlloc(int(left), remains)
		remains = newRemains
		res = fmt.Sprintf("%x%s", alloc, res)
		i--
	}
	return res, remains, nil
}
