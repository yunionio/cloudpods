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
	"reflect"
	"sort"

	"yunion.io/x/pkg/errors"
)

type sJsonNodeValues struct {
	nodeValue    reflect.Value
	nodeValueSet bool
	targetValues []reflect.Value
}

type sJsonUnmarshalSession struct {
	objectMap map[int]*sJsonNodeValues
}

func newJsonUnmarshalSession() *sJsonUnmarshalSession {
	return &sJsonUnmarshalSession{
		objectMap: make(map[int]*sJsonNodeValues),
	}
}

// assignable reports whether the node value can be assigned to the value
// referring to that node
func assignable(nodeValue, refValue reflect.Value) bool {
	if !nodeValue.IsValid() || !refValue.IsValid() {
		return false
	}
	return nodeValue.Type().AssignableTo(refValue.Type())
}

func (s *sJsonUnmarshalSession) saveNodeValue(nodeId int, val reflect.Value) error {
	if nv, ok := s.objectMap[nodeId]; !ok {
		s.objectMap[nodeId] = &sJsonNodeValues{
			nodeValue:    val,
			nodeValueSet: true,
		}
	} else {
		for i := range nv.targetValues {
			if !assignable(val, nv.targetValues[i]) {
				return errors.Wrapf(ErrTypeMismatch, "node id %d vs %s", nodeId, nv.targetValues[i].Type())
			}
		}
		nv.nodeValue = val
		nv.nodeValueSet = true
		for i := range nv.targetValues {
			nv.targetValues[i].Set(val)
		}
		nv.targetValues = nil
	}
	return nil
}

// checkUnboundNodes reports the node ids that were referred to but never
// resolved, which would otherwise leave the referring field untouched
func (s *sJsonUnmarshalSession) checkUnboundNodes() error {
	ids := make([]int, 0)
	for nodeId, nv := range s.objectMap {
		if !nv.nodeValueSet {
			ids = append(ids, nodeId)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Ints(ids)
	return errors.Wrapf(ErrNodeNotFound, "node ids %v", ids)
}

func (s *sJsonUnmarshalSession) setPointerValue(nodeId int, val reflect.Value) error {
	if nv, ok := s.objectMap[nodeId]; ok && nv.nodeValueSet {
		if !assignable(nv.nodeValue, val) {
			return errors.Wrapf(ErrTypeMismatch, "node id %d vs %s", nodeId, val.Type())
		}
		val.Set(nv.nodeValue)
	} else if ok && !nv.nodeValueSet {
		nv.targetValues = append(nv.targetValues, val)
	} else {
		s.objectMap[nodeId] = &sJsonNodeValues{
			nodeValueSet: false,
			targetValues: []reflect.Value{
				val,
			},
		}
	}
	return nil
}
