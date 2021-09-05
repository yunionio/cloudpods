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

package sqlchemy

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type SerializableType struct {
	I  int
	S  string
	IS []int
	SS []string
	M  map[string]int
}

func (t *SerializableType) IsZero() bool {
	return reflect.DeepEqual(t, &SerializableType{})
}

func (t *SerializableType) String() string {
	return jsonutils.Marshal(t).String()
}

func Test_setValueBySQLString(t *testing.T) {
	t.Run("serializable", func(t *testing.T) {
		gotypes.RegisterSerializable(reflect.TypeOf((*SerializableType)(nil)), func() gotypes.ISerializable {
			return &SerializableType{}
		})

		v := &SerializableType{
			I:  100,
			S:  "serializable s value",
			IS: []int{200, 201},
			SS: []string{"s0", "s 1", "s 2"},
			M: map[string]int{
				"k0":  0,
				"k 1": 2,
			},
		}
		s := v.String()

		vv := &SerializableType{}
		vvv := reflect.ValueOf(&vv).Elem()
		if !vvv.CanAddr() {
			t.Fatalf("can not addr")
		}
		if err := setValueBySQLString(vvv, s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(v, vv) {
			t.Fatalf("unequal, want:\n%s\ngot:\n%s", v, vv)
		}
	})

	t.Run("JSONObject", func(t *testing.T) {
		var (
			v     jsonutils.JSONObject
			wantV jsonutils.JSONObject
			s     = `{"i":100,"is":[200,201],"m":{"k 1":2,"k0":0},"s":"serializable s value","ss":["s0","s 1","s 2"]}`
			err   error
		)

		if wantV, err = jsonutils.ParseString(s); err != nil {
			t.Fatalf("parse test json string: %v", err)
		}
		vv := reflect.ValueOf(&v).Elem()
		if err := setValueBySQLString(vv, s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(wantV, v) {
			t.Fatalf("unequal, want:\n%s\ngot:\n%s", v, vv)
		}
	})
}

func TestGetQuoteStringValue(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{
			in:   0,
			want: "0",
		},
		{
			in:   "abc",
			want: "\"abc\"",
		},
		{
			in:   "123\"34",
			want: "\"123\\\"34\"",
		},
	}
	for _, c := range cases {
		got := getQuoteStringValue(c.in)
		if got != c.want {
			t.Errorf("want %s got %s for %s", c.want, got, c.in)
		}
	}
}
