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

package aws

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type AwsXmlRequest struct {
	params map[string]string
	Spec   string
	Local  string
}

func subNodes(p map[string]string) (map[string]map[string]string, map[string][]map[string]string) {
	nodes, array := map[string]map[string]string{}, map[string][]map[string]string{}
	for key, value := range p {
		if !strings.Contains(key, ".") {
			continue
		}
		info := strings.Split(key, ".")
		prefix := info[0]
		if len(info) > 1 {
			idx, err := strconv.Atoi(info[1])
			if err == nil {
				if v, ok := array[prefix]; ok && len(v) == idx || idx == 0 {
					if !ok {
						array[prefix] = []map[string]string{}
					}
					vk := strings.TrimPrefix(key, info[0]+"."+info[1]+".")
					find, index := false, 0
					for adx, value := range array[prefix] {
						for vvk := range value {
							if strings.HasPrefix(vvk, info[2]) {
								find = true
								index = adx
							}
						}
					}
					if !find {
						array[prefix] = append(array[prefix], map[string]string{vk: value})
					} else {
						array[prefix][index][vk] = value
					}
				}
				continue
			}
		}

		_, ok := nodes[prefix]
		if !ok {
			nodes[prefix] = map[string]string{}
		}
		nodes[prefix][strings.TrimPrefix(key, prefix+".")] = value
	}
	return nodes, array
}

func getTokens(p map[string]string, array map[string][]map[string]string, start *xml.StartElement) []xml.Token {
	tokens := []xml.Token{}
	if start != nil {
		tokens = append(tokens, *start)
	}

	for key, value := range p {
		if !strings.Contains(key, ".") {
			t := xml.StartElement{Name: xml.Name{"", key}}
			tokens = append(tokens, t, xml.CharData(value), xml.EndElement{t.Name})
		}
	}

	for key, vv := range array {
		t := xml.StartElement{Name: xml.Name{"", key}}
		tokens = append(tokens, t)
		for _, v := range vv {
			tokens = append(tokens, getTokens(v, nil, nil)...)
		}
		tokens = append(tokens, xml.EndElement{Name: t.Name})
	}

	if len(p)+len(array) == 0 {
		if start != nil {
			tokens = append(tokens, xml.EndElement{Name: start.Name})
		}
		return tokens
	}

	nodes, subArray := subNodes(p)

	tokens = append(tokens, getTokens(nil, subArray, nil)...)

	for k, _nodes := range nodes {
		tokens = append(tokens, getTokens(_nodes, nil, &xml.StartElement{Name: xml.Name{"", k}})...)
	}

	if start != nil {
		tokens = append(tokens, xml.EndElement{Name: start.Name})
	}

	return tokens
}

// StringMap marshals into XML.
func (s AwsXmlRequest) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if start.Name.Local == "AwsXmlRequest" {
		start.Name = xml.Name{
			s.Spec, s.Local,
		}
	}

	tokens := getTokens(s.params, nil, &start)

	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	return e.Flush()
}
