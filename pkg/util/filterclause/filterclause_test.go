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

package filterclause

import (
	"reflect"
	"testing"
)

func TestParseFilterClause(t *testing.T) {
	for _, c := range []string{
		"abc.in(1,2,3)",
		"test.equals(1)",
	} {
		fc := ParseFilterClause(c)
		t.Logf("%s => %s", c, fc.String())
	}
}

func TestParseJointFilterClause(t *testing.T) {
	type args struct {
		jointFilter string
	}
	tests := []struct {
		name string
		args args
		want *SJointFilterClause
	}{
		{
			name: "test parse guestnetworks",
			args: args{
				jointFilter: "guestnetworks.guest_id(id).ip_addr.equals(10.168.222.232)",
			},
			want: &SJointFilterClause{
				SFilterClause: SFilterClause{
					field:    "ip_addr",
					funcName: "equals",
					params:   []string{"10.168.222.232"},
				},
				JointModel: "guestnetworks",
				RelatedKey: "guest_id",
				OriginKey:  "id",
			},
		},
		{
			name: "test parse guestnetworks",
			args: args{
				jointFilter: "networks.id(network_id).name.contains(wp)",
			},
			want: &SJointFilterClause{
				SFilterClause: SFilterClause{
					field:    "name",
					funcName: "contains",
					params:   []string{"wp"},
				},
				JointModel: "networks",
				RelatedKey: "id",
				OriginKey:  "network_id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseJointFilterClause(tt.args.jointFilter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseJointFilterClause() = %v, want %v", got, tt.want)
			}
		})
	}
}
