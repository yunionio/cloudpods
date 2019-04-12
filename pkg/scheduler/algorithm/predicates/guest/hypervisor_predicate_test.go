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

package guest

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func newUnitByHypervisor(hypervisor string) *core.Unit {
	info := &api.SchedInfo{
		ScheduleInput: &schedapi.ScheduleInput{
			ServerConfig: schedapi.ServerConfig{
				ServerConfigs: &computeapi.ServerConfigs{
					Hypervisor: hypervisor,
				},
			},
		},
	}
	return core.NewScheduleUnit(info, nil)
}

type FakeCandidater struct {
	mock.Mock
}

func (c *FakeCandidater) Getter() core.CandidatePropertyGetter {
	return nil
}

func (c *FakeCandidater) IndexKey() string {
	return "fake_id"
}

func (c *FakeCandidater) XGet(key string, kind core.Kind) interface{} {
	args := c.Called()
	return args.String(0)
}

func (c *FakeCandidater) Get(key string) interface{} {
	args := c.Called(key)
	return args.String(0)
}

func (c *FakeCandidater) Type() int {
	return 0
}

func (c *FakeCandidater) GetSchedDesc() *jsonutils.JSONDict {
	return nil
}

func (c *FakeCandidater) GetGuestCount() int64 {
	return 0
}

func (c *FakeCandidater) GetResourceType() string {
	return ""
}

func TestHypervisorPredicate_Execute(t *testing.T) {
	type args struct {
		u *core.Unit
		c core.Candidater
	}

	aliHost := new(FakeCandidater)
	aliHost.On("Get", "HostType").Return(computeapi.HOST_TYPE_ALIYUN)

	tests := []struct {
		name    string
		args    args
		want    bool
		want1   []core.PredicateFailureReason
		wantErr bool
	}{
		{
			name: "hypervisor equals host_type always fits",
			args: args{
				u: newUnitByHypervisor("aliyun"),
				c: aliHost,
			},
			want:    true,
			want1:   nil,
			wantErr: false,
		},
		{
			name: "hypervisor not equals host_type not fit",
			args: args{
				u: newUnitByHypervisor("kvm"),
				c: aliHost,
			},
			want:    false,
			want1:   []core.PredicateFailureReason{predicates.NewUnexceptedResourceError(`host_hypervisor_runtime is 'aliyun', expected 'kvm'`)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &HypervisorPredicate{}
			got, got1, err := f.Execute(tt.args.u, tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("HypervisorPredicate.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HypervisorPredicate.Execute() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("HypervisorPredicate.Execute() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
