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
	"yunion.io/x/pkg/errors"
)

type sNodeReferer struct {
	node     JSONObject
	pointers []*sJSONPointer
}

type sJsonParseSession struct {
	objectMap map[int]*sNodeReferer

	// depth is the current nesting level of the object/array being parsed
	depth int

	// allowNodeReference tells whether the ___jnid_ key and a bare <N>
	// value are read as node references, see ParseTrusted
	allowNodeReference bool
}

func newJsonParseSession(allowNodeReference bool) *sJsonParseSession {
	return &sJsonParseSession{
		objectMap:          make(map[int]*sNodeReferer),
		allowNodeReference: allowNodeReference,
	}
}

// enter records entering one more nesting level, it fails if the nesting
// level exceeds maxParseDepth
func (s *sJsonParseSession) enter() error {
	if s.depth >= maxParseDepth {
		return ErrNestedTooDeep
	}
	s.depth++
	return nil
}

func (s *sJsonParseSession) leave() {
	s.depth--
}

func (s *sJsonParseSession) saveReferer(nodeId int, ptr *sJSONPointer) {
	if nr, ok := s.objectMap[nodeId]; ok {
		nr.pointers = append(nr.pointers, ptr)
	} else {
		s.objectMap[nodeId] = &sNodeReferer{
			pointers: []*sJSONPointer{ptr},
		}
	}
}

func (s *sJsonParseSession) saveNode(nodeId int, node JSONObject) error {
	if nr, ok := s.objectMap[nodeId]; ok {
		if nr.node != nil {
			return errors.Wrapf(ErrDuplicateNodeId, "node id %d", nodeId)
		} else {
			nr.node = node
		}
	} else {
		s.objectMap[nodeId] = &sNodeReferer{
			node:     node,
			pointers: nil,
		}
	}
	return nil
}
