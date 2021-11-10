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
	"fmt"
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
			want: jsonutils.MarshalAll(api.ServerRebuildRootInput{Image: "image"}),
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

type Embeded struct {
}

func (e *Embeded) Method() {
	fmt.Println("Embeded Method")
}

type Struct0 struct {
	Embeded
}

type Struct1 struct {
	Embeded
}

type Struct2 struct {
}

func (e *Struct0) Method() {
	fmt.Println("Struct0 Method")
}

type Top0 struct {
	Struct0
	Struct1
	Struct2
}

type Top1 struct {
	Struct0
	Struct1
	Struct2
}

func (e *Top1) Method() {
	fmt.Println("Top1 Method")
}

func TestFindFunc(t *testing.T) {
	cases := []struct {
		obj  interface{}
		want bool
	}{
		{
			obj:  &Embeded{},
			want: true,
		},
		{
			obj:  &Struct0{},
			want: true,
		},
		{
			obj:  &Struct1{},
			want: true,
		},
		{
			obj:  &Struct2{},
			want: false,
		},
		{
			obj:  &Top0{},
			want: true,
		},
		{
			obj:  &Top1{},
			want: true,
		},
	}
	for _, c := range cases {
		t.Logf("%s is called", reflect.TypeOf(c.obj))
		funcVal, err := findFunc(reflect.ValueOf(c.obj), "Method")
		if funcVal.IsValid() {
			funcVal.Call(nil)
		}
		if (c.want && err != nil) || (!c.want && err == nil) {
			t.Errorf("%s want %v but err==nil %v", reflect.TypeOf(c.obj), c.want, err == nil)
		}
	}
}
