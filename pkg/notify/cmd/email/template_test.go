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

package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	data := struct {
		Name string
		Age  int
	}{"zhengyu", 22}

	jsonStr, _ := json.Marshal(data)

	newData := make(map[string]interface{})
	_ = json.Unmarshal([]byte(jsonStr), &newData)

	tem, _ := template.New("Test").Parse("Name: {{.Name}}; Age: {{.Age}}.")
	buffer := new(bytes.Buffer)
	err := tem.Execute(buffer, newData)
	assert.Nil(t, err)
	assert.Equal(t, "Name: zhengyu; Age: 22.", buffer.String())
}
