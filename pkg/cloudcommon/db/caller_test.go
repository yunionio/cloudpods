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

package db

import (
	"context"
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func Test_valueToJSONObject(t *testing.T) {
	tests := []struct {
		name string
		args interface{}
		want jsonutils.JSONObject
	}{
		{
			name: "json2json",
			args: jsonutils.NewDict(),
			want: jsonutils.NewDict(),
		},
		{
			name: "struct2json",
			args: &api.ServerRebuildRootInput{Image: "image"},
			want: jsonutils.Marshal(api.ServerRebuildRootInput{Image: "image"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValueToJSONObject(reflect.ValueOf(tt.args)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toJSONObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeModel struct{}

func (m *fakeModel) PerformAction(ctx context.Context, input *api.ServerRebuildRootInput) *api.SGuest {
	log.Infof("input: %#v", input)
	out := new(api.SGuest)
	out.Id = input.ImageId
	return out
}

func Test_call(t *testing.T) {
	type args struct {
		modelVal reflect.Value
		fName    string
		inputs   []interface{}
	}

	fModel := new(fakeModel)

	c1Out := new(api.SGuest)
	c1Out.Id = "id"

	tests := []struct {
		name    string
		args    args
		want    []reflect.Value
		wantErr bool
	}{
		{
			name: "input struct",
			args: args{
				modelVal: reflect.ValueOf(fModel),
				fName:    "PerformAction",
				inputs:   []interface{}{context.TODO(), &api.ServerRebuildRootInput{ImageId: "id"}},
			},
			want:    []reflect.Value{reflect.ValueOf(c1Out)},
			wantErr: false,
		},
		{
			name: "input json object",
			args: args{
				modelVal: reflect.ValueOf(fModel),
				fName:    "PerformAction",
				inputs:   []interface{}{context.TODO(), jsonutils.Marshal(api.ServerRebuildRootInput{ImageId: "id"})},
			},
			want:    []reflect.Value{reflect.ValueOf(c1Out)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := callObject(tt.args.modelVal, tt.args.fName, tt.args.inputs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("call() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			log.Infof("out1 %s", jsonutils.Marshal(got[0].Interface()))
			for i := range got {
				gi := got[i].Interface()
				wt := tt.want[i].Interface()
				if !reflect.DeepEqual(gi, wt) {
					t.Errorf("call() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
