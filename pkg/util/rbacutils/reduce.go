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

package rbacutils

import (
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type sRbacNode struct {
	defNode    *sRbacNode
	downStream map[string]*sRbacNode
	result     *TRbacResult
	level      int
}

func newRbacNode(level int) *sRbacNode {
	return &sRbacNode{
		downStream: make(map[string]*sRbacNode),
		level:      level,
	}
}

func (n *sRbacNode) AddRule(rule SRbacRule) {
	n.addRule(rule, levelService)
}

func (n *sRbacNode) addRule(rule SRbacRule, level int) {
	if level <= levelAction || len(rule.Extra) > level-levelExtra {
		if n.result != nil {
			// node is a leaf
			if n.defNode != nil {
				log.Fatalf("illegal state, result != nil and defNode != nil")
			}
			n.defNode = newRbacNode(n.level + 1)
			n.defNode.result = n.result
			n.result = nil
		}
		var key string
		if level == levelService {
			key = rule.Service
		} else if level == levelResource {
			key = rule.Resource
		} else if level == levelAction {
			key = rule.Action
		} else {
			key = rule.Extra[level-levelExtra]
		}
		if len(key) == 0 || key == WILD_MATCH {
			next := n.defNode
			if n.defNode == nil {
				next = newRbacNode(level + 1)
				n.defNode = next
			}
			next.addRule(rule, level+1)
		} else {
			next, ok := n.downStream[key]
			if !ok {
				next = newRbacNode(level + 1)
				n.downStream[key] = next
			}
			next.addRule(rule, level+1)
		}
	} else {
		if n.result != nil {
			log.Warningf("node has been occupide!!!")
		}
		n.result = &rule.Result
	}
}

func (n *sRbacNode) isLeaf() bool {
	return n.result != nil && n.defNode == nil && len(n.downStream) == 0
}

func (n *sRbacNode) reduceDownstream() {
	allowKey := make([]string, 0)
	denyKey := make([]string, 0)
	skipKey := make([]string, 0)
	for k, v := range n.downStream {
		if v.result == nil {
			skipKey = append(skipKey, k)
			continue
		}
		if *v.result == Allow {
			allowKey = append(allowKey, k)
		} else {
			denyKey = append(denyKey, k)
		}
	}
	if len(allowKey)+len(denyKey) > 0 {
		var result TRbacResult
		var keys []string
		if n.defNode != nil {
			if n.defNode.isLeaf() {
				if *n.defNode.result == Allow {
					keys = allowKey
					result = Allow
				} else {
					keys = denyKey
					result = Deny
				}
			}
		} else {
			if len(allowKey) >= len(denyKey) {
				keys = allowKey
				result = Allow
			} else {
				keys = denyKey
				result = Deny
			}
			if len(allowKey) == 0 || len(denyKey) == 0 {
				// reduce whole downStream
				needBranch := false
				if n.level == levelAction {
					sort.Strings(keys)
					if !stringutils2.Contains(stringutils2.SSortedStrings(keys), AllSortedActions) {
						// not a complete set of action, cancel reduce
						needBranch = true
					}
				} else if n.level >= levelExtra {
					if len(keys) <= 1 {
						needBranch = true
					}
				}

				if needBranch {
					keys = nil
					if result == Allow {
						result = Deny
					} else {
						result = Allow
					}
					// add a default branch
					n.defNode = newRbacNode(n.level + 1)
					n.defNode.result = &result
				}
			}
		}
		if len(keys) > 0 {
			for _, k := range keys {
				delete(n.downStream, k)
			}
			if n.defNode == nil {
				n.defNode = newRbacNode(n.level + 1)
			}
			n.defNode.result = &result
		}
	}
	if len(n.downStream) == 0 && n.defNode != nil && n.defNode.isLeaf() {
		n.result = n.defNode.result
		n.defNode = nil
	}
}

func (n *sRbacNode) reduce() {
	if n.defNode != nil {
		n.defNode.reduce()
	}
	for k := range n.downStream {
		n.downStream[k].reduce()
	}
	if n.level == levelService || n.level == levelResource {
		return
	}
	n.reduceDownstream()
}

func (n *sRbacNode) GetRules() []SRbacRule {
	return n.getRules(SRbacRule{Service: WILD_MATCH}, levelService)
}

func (n *sRbacNode) getRules(seed SRbacRule, level int) []SRbacRule {
	result := make([]SRbacRule, 0)
	if n.result != nil {
		rule := seed.clone()
		rule.Result = *n.result
		return []SRbacRule{rule}
	} else {
		if n.defNode != nil {
			rule := seed.clone()
			if level == levelService {
				rule.Service = WILD_MATCH
			} else if level == levelResource {
				rule.Resource = WILD_MATCH
			} else if level == levelAction {
				rule.Action = WILD_MATCH
			} else if level >= levelExtra {
				rule.Extra = append(rule.Extra, WILD_MATCH)
			}
			newRules := n.defNode.getRules(rule, level+1)
			result = append(result, newRules...)
		}
		for k := range n.downStream {
			rule := seed.clone()
			if level == levelService {
				rule.Service = k
			} else if level == levelResource {
				rule.Resource = k
			} else if level == levelAction {
				rule.Action = k
			} else if level >= levelExtra {
				rule.Extra = append(rule.Extra, k)
			}
			newRules := n.downStream[k].getRules(rule, level+1)
			result = append(result, newRules...)
		}
	}
	return result
}

func reduceRules(rules []SRbacRule) []SRbacRule {
	root := newRbacNode(levelService)
	for _, r := range rules {
		root.AddRule(r)
	}
	root.reduce()
	return root.GetRules()
}

func rules2Json(rules []SRbacRule) jsonutils.JSONObject {
	root := newRbacNode(levelService)
	for _, r := range rules {
		root.AddRule(r)
	}
	// root.reduce()
	return root.json()
}

func json2Rules(json jsonutils.JSONObject) ([]SRbacRule, error) {
	root := newRbacNode(levelService)
	err := root.parseJson(json)
	if err != nil {
		return nil, errors.Wrap(err, "root.parseJson")
	}
	// root.reduce()
	return root.GetRules(), nil
}

func (n *sRbacNode) json() jsonutils.JSONObject {
	var result jsonutils.JSONObject
	if n.result != nil {
		return jsonutils.NewString(string(*n.result))
	} else {
		result = jsonutils.NewDict()
		if n.defNode != nil {
			result.(*jsonutils.JSONDict).Add(n.defNode.json(), "*")
		}
		for k := range n.downStream {
			result.(*jsonutils.JSONDict).Add(n.downStream[k].json(), k)
		}
	}
	return result
}

func (n *sRbacNode) parseJson(input jsonutils.JSONObject) error {
	switch val := input.(type) {
	case *jsonutils.JSONString:
		ruleStr, err := val.GetString()
		if err != nil {
			return errors.Wrap(err, "val.GetString")
		}
		var result TRbacResult
		switch ruleStr {
		case string(Allow), string(AdminAllow), string(OwnerAllow), string(UserAllow), string(GuestAllow):
			result = Allow
		default:
			result = Deny
		}
		n.result = &result
	case *jsonutils.JSONDict:
		ruleJsonDict, err := val.GetMap()
		if err != nil {
			return errors.Wrap(err, "val.GetMap")
		}
		for key, ruleJson := range ruleJsonDict {
			if key == WILD_MATCH {
				n.defNode = newRbacNode(n.level + 1)
				err := n.defNode.parseJson(ruleJson)
				if err != nil {
					return errors.Wrap(err, "n.defNode.parseJson")
				}
			} else {
				n.downStream[key] = newRbacNode(n.level + 1)
				err := n.downStream[key].parseJson(ruleJson)
				if err != nil {
					return errors.Wrap(err, "n.downStream[key].parseJson")
				}
			}
		}
	default:
		return errors.Wrap(ErrUnsuportRuleData, input.String())
	}
	return nil
}
