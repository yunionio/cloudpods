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

package version

import (
	"strconv"
	"strings"
)

func less(v1Str, v2Str string) (bool, bool) {
	v1 := strings.Split(v1Str, ".")
	v2 := strings.Split(v2Str, ".")
	var i = 0
	for ; i < len(v2); i++ {
		if i >= len(v1) {
			return true, false
		}
		v, _ := strconv.ParseInt(v2[i], 10, 0)
		compareV, _ := strconv.ParseInt(v1[i], 10, 0)
		if v < compareV {
			return false, false
		} else if compareV < v {
			return true, false
		}
	}
	if i < len(v1)-1 {
		return false, false
	}
	return true, true
}

func LE(v1Str, v2Str string) bool {
	l, _ := less(v1Str, v2Str)
	return l
}

func LT(v1Str, v2Str string) bool {
	l, e := less(v1Str, v2Str)
	return l && !e
}

func GT(v1Str, v2Str string) bool {
	return LT(v2Str, v1Str)
}

func GE(v1Str, v2Str string) bool {
	return LE(v2Str, v1Str)
}
