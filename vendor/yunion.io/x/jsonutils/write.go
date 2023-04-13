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
	"strings"

	"yunion.io/x/pkg/sortedmap"
)

type writeSource interface {
	buildString(sb *strings.Builder)
}

func (this *JSONString) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONValue) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONInt) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONFloat) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONBool) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONDict) buildString(sb *strings.Builder) {
	sb.WriteByte('{')
	var idx = 0
	if this.nodeId > 0 {
		sb.WriteString(quoteString(jsonPointerKey))
		sb.WriteByte(':')
		sb.WriteString(fmt.Sprintf("%d", this.nodeId))
		idx++
	}
	for iter := sortedmap.NewIterator(this.data); iter.HasMore(); iter.Next() {
		k, vinf := iter.Get()
		v := vinf.(JSONObject)
		if idx > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(quoteString(k))
		sb.WriteByte(':')

		v.buildString(sb)
		idx++
	}
	sb.WriteByte('}')
}

func (this *JSONArray) buildString(sb *strings.Builder) {
	sb.WriteByte('[')
	for idx, v := range this.data {
		if idx > 0 {
			sb.WriteByte(',')
		}
		v.buildString(sb)
	}
	sb.WriteByte(']')
}
