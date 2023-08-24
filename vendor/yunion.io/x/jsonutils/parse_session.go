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
)

type sNodeReferer struct {
	node     JSONObject
	pointers []*sJSONPointer
}

type sJsonParseSession struct {
	objectMap map[int]*sNodeReferer
}

func newJsonParseSession() *sJsonParseSession {
	return &sJsonParseSession{
		objectMap: make(map[int]*sNodeReferer),
	}
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

func (s *sJsonParseSession) saveNode(nodeId int, node JSONObject) {
	if nr, ok := s.objectMap[nodeId]; ok {
		if nr.node != nil {
			panic(fmt.Sprintf("nodeId %d alreayd exists: %s != %s", nodeId, nr.node, node))
		} else {
			nr.node = node
		}
	} else {
		s.objectMap[nodeId] = &sNodeReferer{
			node:     node,
			pointers: nil,
		}
	}
}
