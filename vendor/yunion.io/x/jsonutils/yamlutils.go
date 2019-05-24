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

	"github.com/pkg/errors"
)

func parseYAMLArray(lines []string) ([]JSONObject, error) {
	var array = make([]JSONObject, 0)
	var indent = 1
	for indent < len(lines[0]) && lines[0][indent] == ' ' {
		indent++
	}
	var sublines = make([]string, 0)
	for _, line := range lines {
		if indent >= len(line) && len(strings.Trim(line, " ")) == 0 {
			continue
		}
		if line[0] == '-' && len(sublines) > 0 {
			o, e := parseYAMLLines(sublines)
			if e != nil {
				return array, errors.Wrap(e, "parseYAMLLines")
			} else {
				array = append(array, o)
			}
			sublines = make([]string, 0)
		}
		sublines = append(sublines, line[indent:])
	}
	if len(sublines) > 0 {
		o, e := parseYAMLLines(sublines)
		if e != nil {
			return array, errors.Wrap(e, "parseYAMLLines")
		} else {
			array = append(array, o)
		}
	}
	return array, nil
}

func parseYAMLJSONArray(lines []string) (*JSONArray, error) {
	o, e := parseYAMLArray(lines)
	if e != nil {
		return nil, errors.Wrap(e, "parseYAMLArray")
	} else {
		return &JSONArray{data: o}, nil
	}
}

func parseYAMLDict(lines []string) (map[string]JSONObject, error) {
	var dict = make(map[string]JSONObject)
	var i = 0
	for i < len(lines) {
		if len(strings.Trim(lines[i], " ")) == 0 {
			i++
			continue
		}
		var keypos = strings.IndexByte(lines[i], ':')
		if keypos <= 0 {
			return dict, errors.WithMessage(ErrYamlMissingDictKey, lines[i])
		} else {
			key := lines[i][0:keypos]
			val := strings.Trim(lines[i][keypos+1:], " ")

			if len(val) > 0 && val != "|" {
				dict[key] = NewString(val)
				i++
			} else {
				sublines := make([]string, 0)
				j := i + 1
				for j < len(lines) && len(strings.Trim(lines[j], " ")) == 0 {
					sublines = append(sublines, "")
					j++
				}
				if j < len(lines) {
					if lines[j][0] != ' ' {
						return dict, ErrYamlIllFormat // fmt.Errorf("Illformat")
					}

					indent := 0
					for indent < len(lines[j]) && lines[j][indent] == ' ' {
						indent++
					}

					for j < len(lines) {
						if indent >= len(lines[j]) && len(strings.Trim(lines[j], " ")) == 0 {
							sublines = append(sublines, "")
							j++
						} else if indent < len(lines[j]) && len(strings.Trim(lines[j][:indent], " ")) == 0 {
							sublines = append(sublines, lines[j][indent:])
							j++
						} else {
							break
						}
					}
				}
				if val == "|" {
					dict[key] = NewString(strings.Join(sublines, "\n"))
				} else {
					o, e := parseYAMLLines(sublines)
					if e != nil {
						return dict, errors.Wrap(e, "parseYAMLLines")
					}
					dict[key] = o
				}

				i = j
			}
		}
	}
	return dict, nil
}

func parseYAMLJSONDict(lines []string) (*JSONDict, error) {
	o, e := parseYAMLDict(lines)
	if e != nil {
		return nil, errors.Wrap(e, "parseYAMLDict")
	} else {
		return &JSONDict{data: o}, nil
	}
}

func parseYAMLLines(lines []string) (JSONObject, error) {
	var i = 0
	for i < len(lines) && (len(lines[i]) == 0 || lines[i][0] == ' ' || lines[i][0] == '#') {
		i++
	}
	if i >= len(lines) {
		return nil, ErrYamlIllFormat // fmt.Errorf("invalid yaml")
	}
	if lines[i][0] == '-' {
		return parseYAMLJSONArray(lines[i:])
	} else {
		var keypos = strings.IndexByte(lines[i], ':')
		if keypos <= 0 {
			return &JSONString{data: strings.Join(lines[i:], "\n")}, nil
		} else {
			return parseYAMLJSONDict(lines[i:])
		}
	}
}

func ParseYAML(str string) (JSONObject, error) {
	lines := strings.Split(str, "\n")
	return parseYAMLLines(lines)
}

func indentLines(lines []string, is_array bool) []string {
	var ret = make([]string, 0)
	var first_line = true
	for _, line := range lines {
		if is_array && first_line {
			ret = append(ret, "- "+line)
		} else {
			ret = append(ret, "  "+line)
		}
		first_line = false
	}
	return ret
}

func yamlLines2String(o JSONObject) string {
	lines := o.yamlLines()
	return strings.Join(lines, "\n")
}

func (this *JSONString) yamlLines() []string {
	return strings.Split(this.data, "\n")
}

func (this *JSONValue) yamlLines() []string {
	return []string{this.String()}
}

func (this *JSONInt) yamlLines() []string {
	return []string{this.String()}
}

func (this *JSONFloat) yamlLines() []string {
	return []string{this.String()}
}

func (this *JSONBool) yamlLines() []string {
	return []string{this.String()}
}

func (this *JSONArray) yamlLines() []string {
	var ret = make([]string, 0)
	for _, o := range this.data {
		for _, line := range indentLines(o.yamlLines(), true) {
			ret = append(ret, line)
		}
	}
	return ret
}

func (this *JSONDict) yamlLines() []string {
	var ret = make([]string, 0)
	for _, key := range this.SortedKeys() {
		val := this.data[key]
		if val.IsZero() {
			switch val.(type) {
			case *JSONString, *JSONDict, *JSONArray, *JSONValue:
				continue
			}
		}
		lines := val.yamlLines()
		if !val.isCompond() && len(lines) == 1 {
			ret = append(ret, fmt.Sprintf("%s: %s", key, lines[0]))
		} else {
			switch val.(type) {
			case *JSONString:
				ret = append(ret, fmt.Sprintf("%s: |", key))
			default:
				ret = append(ret, fmt.Sprintf("%s:", key))
			}
			for _, line := range indentLines(lines, false) {
				ret = append(ret, line)
			}
		}
	}
	return ret
}

func (this *JSONValue) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONString) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONInt) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONFloat) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONBool) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONArray) YAMLString() string {
	return yamlLines2String(this)
}

func (this *JSONDict) YAMLString() string {
	return yamlLines2String(this)
}
