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

package ovsdb

import (
	"encoding/json"
	"testing"

	"yunion.io/x/pkg/errors"
)

func TestOperationMarshal(t *testing.T) {
	timeout := 0
	cases := []struct {
		name string
		op   Operation
		want string
	}{
		{
			name: "select all rows",
			op:   SelectOp("Logical_Switch", Where()),
			want: `{"op":"select","table":"Logical_Switch","where":[]}`,
		},
		{
			name: "select by uuid",
			op:   SelectOp("Logical_Switch", WhereUuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"), "name"),
			want: `{"op":"select","table":"Logical_Switch","columns":["name"],` +
				`"where":[["_uuid","==",["uuid","ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"]]]}`,
		},
		{
			name: "insert",
			op:   InsertOp("Logical_Switch", Row{"name": "ls0"}, "row0"),
			want: `{"op":"insert","table":"Logical_Switch","row":{"name":"ls0"},"uuid-name":"row0"}`,
		},
		{
			name: "insert without a name",
			op:   InsertOp("Logical_Switch", Row{"name": "ls0"}, ""),
			want: `{"op":"insert","table":"Logical_Switch","row":{"name":"ls0"}}`,
		},
		{
			name: "update",
			op: UpdateOp("Logical_Switch", Where(Condition{"name", FnEq, "ls0"}),
				Row{"other_config": NewMapStringString(map[string]string{"mcast_snoop": "true"})}),
			want: `{"op":"update","table":"Logical_Switch",` +
				`"row":{"other_config":["map",[["mcast_snoop","true"]]]},` +
				`"where":[["name","==","ls0"]]}`,
		},
		{
			name: "mutate",
			op: MutateOp("Logical_Switch", WhereUuidRef("@row0"),
				Mutation{"ports", MutInsert, Set(UuidValues([]string{"@row1"}))}),
			want: `{"op":"mutate","table":"Logical_Switch",` +
				`"mutations":[["ports","insert",["set",[["named-uuid","row1"]]]]],` +
				`"where":[["_uuid","==",["named-uuid","row0"]]]}`,
		},
		{
			name: "delete",
			op:   DeleteOp("Logical_Switch_Port", Where(Condition{"name", FnEq, "lsp0"})),
			want: `{"op":"delete","table":"Logical_Switch_Port","where":[["name","==","lsp0"]]}`,
		},
		{
			name: "wait",
			op: WaitOp("Logical_Switch", Where(Condition{"name", FnEq, "ls0"}),
				[]string{"name"}, "==", []Row{{"name": "ls0"}}, &timeout),
			want: `{"op":"wait","table":"Logical_Switch","rows":[{"name":"ls0"}],` +
				`"columns":["name"],"timeout":0,"where":[["name","==","ls0"]],"until":"=="}`,
		},
		{
			name: "comment",
			op:   CommentOp("claim vpc 0"),
			want: `{"op":"comment","comment":"claim vpc 0"}`,
		},
		{
			name: "assert",
			op:   AssertOp("ovn_nb_lock"),
			want: `{"op":"assert","lock":"ovn_nb_lock"}`,
		},
		{
			name: "commit",
			op:   CommitOp(true),
			want: `{"op":"commit","durable":true}`,
		},
		{
			name: "abort",
			op:   AbortOp(),
			want: `{"op":"abort"}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.op)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("got  %s\nwant %s", string(got), c.want)
			}
		})
	}
}

func TestOperationResultUnmarshal(t *testing.T) {
	data := `[
		{"uuid":["uuid","ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"]},
		{"count":2},
		{"rows":[{"name":"ls0"}]},
		{"error":"constraint violation","details":"duplicate name"}
	]`
	var results []OperationResult
	if err := json.Unmarshal([]byte(data), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("want 4 results, got %d", len(results))
	}
	if results[0].Uuid != "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c" {
		t.Errorf("uuid: got %q", results[0].Uuid)
	}
	if results[1].Count != 2 {
		t.Errorf("count: got %d", results[1].Count)
	}
	if len(results[2].Rows) != 1 || results[2].Rows[0]["name"] != "ls0" {
		t.Errorf("rows: got %#v", results[2].Rows)
	}
	if results[3].Error != "constraint violation" {
		t.Errorf("error: got %q", results[3].Error)
	}
}

func TestCheckOperationResults(t *testing.T) {
	ops := []Operation{
		InsertOp("Logical_Switch", Row{"name": "ls0"}, "row0"),
		InsertOp("Logical_Switch_Port", Row{"name": "lsp0"}, "row1"),
	}

	t.Run("all good", func(t *testing.T) {
		results := []OperationResult{{}, {}}
		if err := CheckOperationResults(results, ops); err != nil {
			t.Errorf("got %v", err)
		}
	})

	t.Run("operation error", func(t *testing.T) {
		results := []OperationResult{{}, {Error: "constraint violation", Details: "duplicate name"}}
		err := CheckOperationResults(results, ops)
		if err == nil {
			t.Fatalf("want an error")
		}
		if errors.Cause(err) != ErrOperation {
			t.Errorf("want ErrOperation, got %v", err)
		}
	})

	t.Run("short result means the rest was never attempted", func(t *testing.T) {
		results := []OperationResult{{}}
		if err := CheckOperationResults(results, ops); err == nil {
			t.Errorf("want an error")
		}
	})

	t.Run("trailing transaction error", func(t *testing.T) {
		// rfc7047 5.2, one extra element may describe a failure of the
		// transaction as a whole
		results := []OperationResult{{}, {}, {Error: "referential integrity violation"}}
		err := CheckOperationResults(results, ops)
		if err == nil {
			t.Fatalf("want an error")
		}
		if errors.Cause(err) != ErrOperation {
			t.Errorf("want ErrOperation, got %v", err)
		}
	})
}
