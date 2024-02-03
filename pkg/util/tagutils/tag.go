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

package tagutils

import (
	"fmt"
	"strings"
)

const (
	NoValue  = "___no_value__"
	AnyValue = ""
)

type STag struct {
	// 标签Kye
	Key string `json:"key"`
	// 标签Value
	Value string `json:"value"`
}

func (t STag) String() string {
	return fmt.Sprintf("%s=%s", t.Key, t.Value)
}

func (t STag) KeyPrefix() string {
	commaPos := strings.Index(t.Key, ":")
	if commaPos > 0 {
		return t.Key[:commaPos]
	}
	return ""
}

func Compare(t1, t2 STag) int {
	if t1.Key < t2.Key {
		return -1
	} else if t1.Key > t2.Key {
		return 1
	}
	if t1.Value != t2.Value {
		if t1.Value == AnyValue {
			return -1
		} else if t2.Value == AnyValue {
			return 1
		}
		if t1.Value == NoValue {
			return 1
		} else if t2.Value == NoValue {
			return -1
		}
		if t1.Value < t2.Value {
			return -1
		} else if t1.Value > t2.Value {
			return 1
		}
	}
	return 0
}
