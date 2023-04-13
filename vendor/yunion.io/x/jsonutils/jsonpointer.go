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
	"reflect"
	"strings"
)

const (
	jsonPointerKey = "___jnid_"
)

type sJSONPointer struct {
	JSONValue
	nodeId int
}

func (s *sJsonMarshalSession) newJsonPointer(inf interface{}) *sJSONPointer {
	s.nodeIndex++
	jsonPtr := &sJSONPointer{
		nodeId: s.nodeIndex,
	}
	s.objectMap[inf] = jsonPtr
	s.nodeMap[jsonPtr.nodeId] = &sJsonPointerRefCount{refCnt: 1}
	return jsonPtr
}

func (s *sJsonMarshalSession) setJsonObject(ptr *sJSONPointer, obj JSONObject) {
	if jsonDict, ok := obj.(*JSONDict); ok {
		if jr, ok := s.nodeMap[ptr.nodeId]; ok {
			jr.node = jsonDict
		} else {
			panic(fmt.Sprintf("nodeId %d should exists!", ptr.nodeId))
		}
	}
}

func (s *sJsonMarshalSession) addPointerReferer(ptr *sJSONPointer) {
	if jr, ok := s.nodeMap[ptr.nodeId]; ok {
		jr.refCnt++
	} else {
		panic(fmt.Sprintf("fail to find nodeId %d???", ptr.nodeId))
	}
}

func (s *sJsonMarshalSession) setAllNodeId() {
	for nodeId, refer := range s.nodeMap {
		if refer.node != nil && refer.refCnt > 1 {
			refer.node.setNodeId(nodeId)
		}
	}
}

func (ptr *sJSONPointer) String() string {
	return fmt.Sprintf("<%d>", ptr.nodeId)
}

func (ptr *sJSONPointer) PrettyString() string {
	return ptr.String()
}

func (ptr *sJSONPointer) buildString(sb *strings.Builder) {
	sb.WriteString(ptr.String())
}

func (dict *JSONDict) setNodeId(nodeId int) {
	dict.nodeId = nodeId
}

func (ptr *sJSONPointer) unmarshalValue(s *sJsonUnmarshalSession, val reflect.Value) error {
	return s.setPointerValue(ptr.nodeId, val)
}
