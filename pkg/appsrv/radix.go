package appsrv

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/log"
)

type RadixNode struct {
	data        interface{}
	stringNodes map[string]*RadixNode
	regexpNodes map[string]*RadixNode
	segNames    map[int]string
}

func NewRadix() *RadixNode {
	return &RadixNode{
		data:        nil,
		stringNodes: make(map[string]*RadixNode, 0),
		regexpNodes: make(map[string]*RadixNode, 0),
		segNames:    nil,
	}
}

func isRegexSegment(seg string) bool {
	if len(seg) > 2 && seg[0] == '<' && seg[len(seg)-1] == '>' {
		return true
	} else {
		return false
	}
}

func (r *RadixNode) Add(segments []string, data interface{}) error {
	err := r.add(segments, data, 1, nil)
	if err != nil {
		return fmt.Errorf("Add Node error: %s %s", err, strings.Join(segments, "/"))
	}
	return nil
}

func (r *RadixNode) add(segments []string, data interface{}, depth int, segNames map[int]string) error {
	if len(segments) == 0 {
		if r.data != nil {
			return fmt.Errorf("Duplicate data for node")
		} else {
			r.data = data
			r.segNames = segNames
			return nil
		}
	} else {
		var nextNode *RadixNode
		if isRegexSegment(segments[0]) {
			var (
				regStr     string
				segName    string
				segStr     = segments[0][1 : len(segments[0])-1]
				splitIndex = strings.IndexByte(segStr, ':')
			)

			if splitIndex < 0 {
				regStr = ".*" // match anything
				segName = "<" + segStr + ">"
			} else {
				regStr = segStr[splitIndex+1:]
				segName = "<" + segStr[0:splitIndex] + ">"
			}

			if segNames == nil {
				segNames = make(map[int]string, 0)
			}
			segNames[depth-1] = segName

			if node, ok := r.regexpNodes[regStr]; ok {
				nextNode = node
			} else {
				nextNode = NewRadix()
				r.regexpNodes[regStr] = nextNode
			}
		} else {
			if node, ok := r.stringNodes[segments[0]]; ok {
				nextNode = node
			} else {
				nextNode = NewRadix()
				r.stringNodes[segments[0]] = nextNode
			}
		}
		return nextNode.add(segments[1:], data, depth+1, segNames)
	}
}

func (r *RadixNode) Match(segments []string, params map[string]string) interface{} {
	node, _ := r.match(segments)
	if node == nil {
		return nil
	}
	for index, segName := range node.segNames {
		params[segName] = segments[index]
	}
	return node.data
}

func (r *RadixNode) match(segments []string) (*RadixNode, bool) {
	if len(segments) == 0 {
		return r, true
	} else if len(r.stringNodes) == 0 && len(r.regexpNodes) == 0 {
		return r, false
	}

	if node, ok := r.stringNodes[segments[0]]; ok {
		if rnode, _ := node.match(segments[1:]); rnode != nil && rnode.data != nil {
			return rnode, true
		}
	}

	var nodeTmp *RadixNode
	for regstr, node := range r.regexpNodes {
		if regexp.MustCompile(regstr).MatchString(segments[0]) {
			if rnode, fullMatch := node.match(segments[1:]); rnode != nil && rnode.data != nil {
				if fullMatch {
					return rnode, fullMatch
				} else {
					if nodeTmp != nil {
						log.Errorf("segments %v match mutil node", segments)
						continue
					}
					nodeTmp = rnode
				}
			}
		}
	}

	if nodeTmp != nil {
		return nodeTmp, false
	} else {
		return r, false
	}
}

func (r *RadixNode) Walk(f func(spath string, data interface{})) {
	r.walk("/", f)
}

func (r *RadixNode) walk(fullPath string, f func(spath string, data interface{})) {
	if r.data != nil {
		f(fullPath, r.data)
	}

	for key, node := range r.stringNodes {
		curPath := path.Join(fullPath, key)
		node.walk(curPath, f)
	}
	for key, node := range r.regexpNodes {
		curPath := path.Join(fullPath, key)
		node.walk(curPath, f)
	}
}
