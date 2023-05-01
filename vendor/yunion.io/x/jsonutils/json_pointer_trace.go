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
)

type sJsonPointerNode struct {
	pointer *sJSONPointer
	inf     interface{}
	refcnt  int
}

func (n *sJsonPointerNode) String() string {
	return fmt.Sprintf("<%d>", n.pointer.nodeId)
}

type sJsonPointerTrace struct {
	trace []*sJsonPointerNode
}

func newJsonPointerTrace() *sJsonPointerTrace {
	return &sJsonPointerTrace{
		trace: make([]*sJsonPointerNode, 0, 10),
	}
}

func (t *sJsonPointerTrace) push(inf interface{}, ptr *sJSONPointer) *sJsonPointerNode {
	node := &sJsonPointerNode{
		pointer: ptr,
		inf:     inf,
		refcnt:  1,
	}
	t.trace = append(t.trace, node)
	// log.Debugf("push: %s", t.String())
	return node
}

func (t *sJsonPointerTrace) find(inf interface{}) *sJsonPointerNode {
	var ret *sJsonPointerNode
	for i := range t.trace {
		if t.trace[i].inf == inf {
			ret = t.trace[i]
		}
		if ret != nil {
			t.trace[i].refcnt++
		}
	}
	return ret
}

func (t *sJsonPointerTrace) pop() {
	if len(t.trace) > 0 && t.trace[len(t.trace)-1].refcnt == 1 {
		t.trace = t.trace[0 : len(t.trace)-1]
	}
	// log.Debugf("pop: %s", t.String())
}

func (t *sJsonPointerTrace) String() string {
	buf := &strings.Builder{}
	for i := range t.trace {
		buf.WriteString(t.trace[i].String())
	}
	return buf.String()
}
