package appsrv

import (
	"fmt"
	"strings"
)

type RadixNode struct {
	data      interface{}
	fullPath  []string
	next      []*RadixNode
	parent    *RadixNode
	matchNext *RadixNode
	// matchTable []string
	segment string
}

func NewRadix() *RadixNode {
	return &RadixNode{data: nil,
		fullPath:  nil,
		next:      make([]*RadixNode, 0),
		matchNext: nil,
		parent:    nil,
		segment:   ""}
}

func (r *RadixNode) String() string {
	return strings.Join(r.Segments(), "/")
}

func (r *RadixNode) Segments() []string {
	return r.appendSegment(make([]string, 0))
}

func (r *RadixNode) appendSegment(segs []string) []string {
	if r.parent != nil {
		segs = r.parent.appendSegment(segs)
	}
	if len(r.segment) > 0 {
		segs = append(segs, r.segment)
	}
	return segs
}

func isMatchSegment(seg string) bool {
	if len(seg) > 2 && seg[0] == '<' && seg[len(seg)-1] == '>' {
		return true
	} else {
		return false
	}
}

func (r *RadixNode) Add(segments []string, data interface{}) error {
	return r.add(segments, segments, data)
}

func (r *RadixNode) add(path []string, segments []string, data interface{}) error {
	// log.Debugf("add %#v %#v", path, segments)

	if len(segments) == 0 {
		if r.data != nil {
			return fmt.Errorf("Duplicate data for node %s", r.String())
		} else {
			r.data = data
			r.fullPath = make([]string, len(path))
			for i := 0; i < len(path); i += 1 {
				r.fullPath[i] = path[i]
			}
			return nil
		}
	}
	var nextNode *RadixNode = nil
	if isMatchSegment(segments[0]) {
		if r.matchNext != nil {
			/* if r.matchNext.segment != segments[0] {
			    return fmt.Errorf("%s has been registered, %s conflict with %s", r.matchNext.String(), r.matchNext.segment, segments[0])
			} */
			nextNode = r.matchNext
			// nextNode.matchTable = append(nextNode.matchTable, segments[0])
		} else {
			nextNode = NewRadix()
			nextNode.segment = "<*>"
			nextNode.parent = r
			// nextNode.matchTable = []string{segments[0]}
			r.matchNext = nextNode
		}
	} else {
		for i := 0; i < len(r.next); i += 1 {
			if r.next[i].segment == segments[0] {
				nextNode = r.next[i]
				break
			}
		}
		if nextNode == nil {
			nextNode = NewRadix()
			nextNode.segment = segments[0]
			nextNode.parent = r
			r.next = append(r.next, nextNode)
		}
	}
	return nextNode.add(path, segments[1:], data)
}

func (r *RadixNode) Match(segments []string, params map[string]string) interface{} {
	data, allPaths := r.match(segments)
	// log.Debugf("%#v", allPaths)
	for i := 0; i < len(segments); i += 1 {
		for j := 0; j < len(allPaths); j += 1 {
			if i < len(allPaths[j]) && isMatchSegment(allPaths[j][i]) {
				params[allPaths[j][i]] = segments[i]
			}
		}
	}
	return data
}

func (r *RadixNode) match(segments []string) (interface{}, [][]string) {
	if len(segments) == 0 {
		return r.data, r.getAllFullPaths()
	} else {
		var retData interface{} = nil
		var retPath [][]string = nil
		exactMatch := false
		for _, node := range r.next {
			if node.segment == segments[0] {
				retData, retPath = node.match(segments[1:])
				if retData != nil {
					exactMatch = true
				}
				break
			}
		}
		if retData != nil {
			return retData, retPath
		} else {
			if !exactMatch && r.matchNext != nil {
				retData, retPath = r.matchNext.match(segments[1:])
				return retData, retPath
			} else {
				return r.data, r.getAllFullPaths()
			}
		}
	}
}

func (r *RadixNode) Walk(f func(path string, data interface{})) {
	if r.data != nil {
		f(r.String(), r.data)
	}
	for _, node := range r.next {
		node.Walk(f)
	}
	if r.matchNext != nil {
		r.matchNext.Walk(f)
	}
}

func (r *RadixNode) getAllFullPaths() [][]string {
	if r.fullPath != nil {
		return [][]string{r.fullPath}
	} else {
		ret := make([][]string, 0)
		for _, node := range r.next {
			fp := node.getAllFullPaths()
			ret = append(ret, fp...)
		}
		if r.matchNext != nil {
			fp := r.matchNext.getAllFullPaths()
			ret = append(ret, fp...)
		}
		return ret
	}
}
