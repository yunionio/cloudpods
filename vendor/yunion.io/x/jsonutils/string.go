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

package jsonutils

import (
	"fmt"
	"strconv"
	"strings"
)

func (this *JSONString) String() string {
	return quoteString(this.data)
}

func (this *JSONValue) String() string {
	return "null"
}

func (this *JSONInt) String() string {
	return fmt.Sprintf("%d", this.data)
}

func (this *JSONFloat) String() string {
	if this.bit != 32 && this.bit != 64 {
		this.bit = 64
	}
	return strconv.FormatFloat(this.data, 'f', -1, this.bit)
}

func (this *JSONBool) String() string {
	if this.data {
		return "true"
	} else {
		return "false"
	}
}

func (this *JSONDict) String() string {
	sb := &strings.Builder{}
	this.buildString(sb)
	return sb.String()
}

func (this *JSONArray) String() string {
	sb := &strings.Builder{}
	this.buildString(sb)
	return sb.String()
}
