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

package k8s

import (
	"reflect"
	"testing"
)

func Test_parsePolicyRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    string
		want    *PolicyRule
		wantErr bool
	}{
		{
			name: "apps/v1:deployments,daemonsets:get,watch",
			rule: "apps/v1:deployments,daemonsets:get,watch",
			want: &PolicyRule{
				APIGroups: []string{"apps/v1"},
				Resources: []string{"deployments", "daemonsets"},
				Verbs:     []string{"get", "watch"},
			},
			wantErr: false,
		},
		{
			name: "*:*:get,watch",
			rule: "*:*:get,watch",
			want: &PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "watch"},
			},
			wantErr: false,
		},
		{
			name: ":pods,configmaps:*",
			rule: ":pods,configmaps:*",
			want: &PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods", "configmaps"},
				Verbs:     []string{"*"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePolicyRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePolicyRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePolicyRule() = %v, want %v", got, tt.want)
			}
		})
	}
}
