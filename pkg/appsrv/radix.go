package appsrv

import (
	"fmt"
	"strings"
)

type RadixNode struct {
	data       interface{}
	next       []*RadixNode
	parent     *RadixNode
	matchNext  *RadixNode
	matchTable []string
	segment    string
}

func NewRadix() *RadixNode {
	return &RadixNode{data: nil,
		next:       make([]*RadixNode, 0),
		matchNext:  nil,
		matchTable: nil,
		parent:     nil,
		segment:    ""}
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
	if len(segments) == 0 {
		if r.data != nil {
			return fmt.Errorf("Duplicate data for node %s", r.String())
		} else {
			r.data = data
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
			nextNode.matchTable = append(nextNode.matchTable, segments[0])
		} else {
			nextNode = NewRadix()
			nextNode.segment = "<*>"
			nextNode.parent = r
			nextNode.matchTable = []string{segments[0]}
			r.matchNext = nextNode
		}
	} else {
		for _, node := range r.next {
			if node.segment == segments[0] {
				nextNode = node
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
	return nextNode.Add(segments[1:], data)
}

func (r *RadixNode) Match(segments []string, params map[string]string) interface{} {
	if len(segments) == 0 {
		return r.data
	} else {
		var ret interface{} = nil
		exactMatch := false
		for _, node := range r.next {
			if node.segment == segments[0] {
				ret = node.Match(segments[1:], params)
				if ret != nil {
					// log.Debugf("Match %s ret %#v", node.segment, ret)
					exactMatch = true
				} else {
					// log.Debugf("No match %s ret %#v", node.segment, ret)
				}
				break
			}
		}
		if ret != nil {
			return ret
		} else {
			if !exactMatch && r.matchNext != nil {
				ret = r.matchNext.Match(segments[1:], params)
				if ret != nil {
					for _, segname := range r.matchNext.matchTable {
						if _, ok := params[segname]; !ok {
							params[segname] = segments[0]
						}
					}
				}
				return ret
			} else {
				return r.data
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
