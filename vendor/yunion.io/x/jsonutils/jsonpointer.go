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
	"time"
)

const (
	jsonPointerKey = "___jnid_"
)

type sJSONPointer struct {
	node   JSONObject
	nodeId int
}

var _ JSONObject = (*sJSONPointer)(nil)

func (s *sJsonMarshalSession) newJsonPointer(inf interface{}) *sJSONPointer {
	s.nodeIndex++
	jsonPtr := &sJSONPointer{
		nodeId: s.nodeIndex,
	}
	s.nodeMap[s.nodeIndex] = s.objectTrace.push(inf, jsonPtr)
	return jsonPtr
}

func (s *sJsonMarshalSession) setJsonObject(ptr *sJSONPointer, obj JSONObject) {
	if _, ok := obj.(*JSONDict); ok {
		ptr.node = obj
	}
	s.objectTrace.pop()
}

func (s *sJsonMarshalSession) setAllNodeId() {
	for nodeId, jsonPtrNode := range s.nodeMap {
		if jsonPtrNode.refcnt > 1 && jsonPtrNode.pointer.node != nil {
			jsonPtrNode.pointer.node.(*JSONDict).setNodeId(nodeId)
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

func (ptr *sJSONPointer) IsZero() bool {
	if ptr.node != nil {
		return ptr.node.IsZero()
	}
	return true
}

func (ptr *sJSONPointer) parse(s *sJsonParseSession, str []byte, offset int) (int, error) {
	// null ops
	return -1, nil
}

func (ptr *sJSONPointer) prettyString(level int) string {
	return jsonPrettyString(ptr, level)
}

func (ptr *sJSONPointer) YAMLString() string {
	return ptr.String()
}

func (ptr *sJSONPointer) QueryString() string {
	return ""
}

func (ptr *sJSONPointer) _queryString(key string) string {
	return ""
}

func (ptr *sJSONPointer) getNode() JSONObject {
	if ptr.node != nil {
		return ptr.node
	}
	return JSONNull
}

func (ptr *sJSONPointer) Contains(keys ...string) bool {
	return ptr.getNode().Contains(keys...)
}

func (ptr *sJSONPointer) ContainsIgnoreCases(keys ...string) bool {
	return ptr.getNode().ContainsIgnoreCases(keys...)
}

func (ptr *sJSONPointer) Get(keys ...string) (JSONObject, error) {
	return ptr.getNode().Get(keys...)
}

func (ptr *sJSONPointer) GetIgnoreCases(keys ...string) (JSONObject, error) {
	return ptr.getNode().GetIgnoreCases(keys...)
}

func (ptr *sJSONPointer) GetAt(i int, keys ...string) (JSONObject, error) {
	return ptr.getNode().GetAt(i, keys...)
}

func (ptr *sJSONPointer) Int(keys ...string) (int64, error) {
	return ptr.getNode().Int(keys...)
}

func (ptr *sJSONPointer) Float(keys ...string) (float64, error) {
	return ptr.getNode().Float(keys...)
}

func (ptr *sJSONPointer) Bool(keys ...string) (bool, error) {
	return ptr.getNode().Bool(keys...)
}

func (ptr *sJSONPointer) GetMap(keys ...string) (map[string]JSONObject, error) {
	return ptr.getNode().GetMap(keys...)
}

func (ptr *sJSONPointer) GetArray(keys ...string) ([]JSONObject, error) {
	return ptr.getNode().GetArray(keys...)
}

func (ptr *sJSONPointer) GetTime(keys ...string) (time.Time, error) {
	return ptr.getNode().GetTime(keys...)
}

func (ptr *sJSONPointer) GetString(keys ...string) (string, error) {
	return ptr.getNode().GetString(keys...)
}

func (ptr *sJSONPointer) Unmarshal(obj interface{}, keys ...string) error {
	return ptr.getNode().Unmarshal(obj, keys...)
}

func (ptr *sJSONPointer) Equals(obj JSONObject) bool {
	return ptr.getNode().Equals(obj)
}

func (ptr *sJSONPointer) Interface() interface{} {
	return ptr.getNode().Interface()
}

func (ptr *sJSONPointer) isCompond() bool {
	return ptr.getNode().isCompond()
}
