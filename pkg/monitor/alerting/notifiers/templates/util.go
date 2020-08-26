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

package templates

import (
	"bytes"
	"fmt"
	"html/template"
	t_template "text/template"
)

func CompileTemplateFromMapHtml(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := template.Must(template.New("commpiled_template").Funcs(
		template.FuncMap{
			"FormateFloat": FormateFloat,
		}).Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", err
	}
	return out.String(), nil
}

func CompileTEmplateFromMapText(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := t_template.Must(t_template.New("commpiled_template").Funcs(
		t_template.FuncMap{
			"FormateFloat": FormateFloat,
		}).Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", err
	}
	return out.String(), nil
}

func FormateFloat(f *float64) string {
	return fmt.Sprintf("%f", *f)
}
