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

package models

import (
	"context"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestAnsiblePlaybookReferenceValidateUpdateData(t *testing.T) {
	ar := &SAnsiblePlaybookReference{}
	ctx := context.Background()
	body := func(data map[string]interface{}) *jsonutils.JSONDict {
		return jsonutils.Marshal(data).(*jsonutils.JSONDict)
	}

	cases := []struct {
		name    string
		data    *jsonutils.JSONDict
		wantErr bool
	}{
		{
			name:    "playbook params only",
			data:    body(map[string]interface{}{"playbook_params": map[string]interface{}{"a": "b"}}),
			wantErr: false,
		},
		{
			name:    "playbook params with another field",
			data:    body(map[string]interface{}{"playbook_params": map[string]interface{}{"a": "b"}, "name": "x"}),
			wantErr: true,
		},
		{
			name:    "another field only",
			data:    body(map[string]interface{}{"name": "x"}),
			wantErr: true,
		},
		{
			name:    "empty body",
			data:    body(map[string]interface{}{}),
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ar.ValidateUpdateData(ctx, nil, nil, c.data)
			if c.wantErr != (err != nil) {
				t.Fatalf("wantErr %v, got %v", c.wantErr, err)
			}
		})
	}
}
