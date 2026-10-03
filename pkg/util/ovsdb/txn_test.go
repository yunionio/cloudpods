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
	"context"
	"encoding/json"
	"testing"

	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/pkg/errors"
)

func newTxnClient(t *testing.T) (*Client, *fakeServer) {
	t.Helper()
	cli, srv := newFakeServer(t, nil)
	if _, err := cli.GetSchema(context.Background(), testDb); err != nil {
		t.Fatalf("get_schema: %v", err)
	}
	return cli, srv
}

// txnOpsJSON returns the operations of a transaction as they go on the wire
func txnOpsJSON(t *testing.T, txn *Txn) string {
	t.Helper()
	if err := txn.Err(); err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	data, err := json.Marshal(txn.Ops())
	if err != nil {
		t.Fatalf("marshal ops: %v", err)
	}
	return string(data)
}

func TestTxnInsertWithReference(t *testing.T) {
	cli, _ := newTxnClient(t)

	txn := cli.Txn(testDb)
	lsRef := txn.Insert(&ovn_nb.LogicalSwitch{Name: "ls0"})
	lspRef := txn.Insert(&ovn_nb.LogicalSwitchPort{Name: "lsp0", Type: "router"})
	txn.AddRefWhere("Logical_Switch", WhereUuidRef(lsRef), "ports", lspRef)

	if lsRef != "@row1" || lspRef != "@row2" {
		t.Fatalf("refs: got %q %q", lsRef, lspRef)
	}
	want := `[` +
		`{"op":"insert","table":"Logical_Switch","row":{"name":"ls0"},"uuid-name":"row1"},` +
		`{"op":"insert","table":"Logical_Switch_Port","row":{"name":"lsp0","type":"router"},"uuid-name":"row2"},` +
		`{"op":"mutate","table":"Logical_Switch","mutations":[["ports","insert",["set",[["named-uuid","row2"]]]]],` +
		`"where":[["_uuid","==",["named-uuid","row1"]]]}` +
		`]`
	if got := txnOpsJSON(t, txn); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestTxnAddDelRef(t *testing.T) {
	cli, _ := newTxnClient(t)
	ls := &ovn_nb.LogicalSwitch{
		Uuid: "0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c",
		Name: "ls0",
	}

	txn := cli.Txn(testDb)
	txn.AddRef(ls, "ports", "1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d")
	txn.DelRef(ls, "ports", "2c5d3c80-3f4e-6b7d-9cae-7f605b4c3d2e")
	want := `[` +
		`{"op":"mutate","table":"Logical_Switch",` +
		`"mutations":[["ports","insert",["set",[["uuid","1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"]]]]],` +
		`"where":[["_uuid","==",["uuid","0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"]]]},` +
		`{"op":"mutate","table":"Logical_Switch",` +
		`"mutations":[["ports","delete",["set",[["uuid","2c5d3c80-3f4e-6b7d-9cae-7f605b4c3d2e"]]]]],` +
		`"where":[["_uuid","==",["uuid","0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"]]]}` +
		`]`
	if got := txnOpsJSON(t, txn); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestTxnUpdateAndDelete(t *testing.T) {
	cli, _ := newTxnClient(t)
	lsp := &ovn_nb.LogicalSwitchPort{
		Uuid:      "1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d",
		Name:      "lsp0",
		Addresses: []string{"00:11:22:33:44:55 10.0.0.2"},
	}

	txn := cli.Txn(testDb)
	txn.UpdateColumns(lsp, "addresses")
	txn.Delete(lsp)
	want := `[` +
		`{"op":"update","table":"Logical_Switch_Port",` +
		`"row":{"addresses":["set",["00:11:22:33:44:55 10.0.0.2"]]},` +
		`"where":[["_uuid","==",["uuid","1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"]]]},` +
		`{"op":"delete","table":"Logical_Switch_Port",` +
		`"where":[["_uuid","==",["uuid","1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"]]]}` +
		`]`
	if got := txnOpsJSON(t, txn); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestTxnUpdateNeedsUuid(t *testing.T) {
	cli, _ := newTxnClient(t)
	txn := cli.Txn(testDb)
	txn.Update(&ovn_nb.LogicalSwitchPort{Name: "lsp0"})
	if txn.Err() == nil {
		t.Fatalf("want an error for a row without a uuid")
	}
	if _, err := txn.Commit(context.Background()); err == nil {
		t.Errorf("commit should report the error the build met")
	}
}

func TestTxnUpdateColumnsUnknownColumn(t *testing.T) {
	cli, _ := newTxnClient(t)
	lsp := &ovn_nb.LogicalSwitchPort{Uuid: "1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d", Name: "lsp0"}
	txn := cli.Txn(testDb)
	txn.UpdateColumns(lsp, "no_such_column")
	if errors.Cause(txn.Err()) != ErrSchema {
		t.Errorf("want ErrSchema, got %v", txn.Err())
	}
}

func TestTxnWithoutSchema(t *testing.T) {
	cli, _ := newFakeServer(t, nil)
	txn := cli.Txn(testDb)
	if errors.Cause(txn.Err()) != ErrSchema {
		t.Errorf("want ErrSchema, got %v", txn.Err())
	}
	// a failed build adds nothing and never reaches the remote
	txn.Insert(&ovn_nb.LogicalSwitch{Name: "ls0"})
	if txn.Len() != 0 {
		t.Errorf("want no operations, got %d", txn.Len())
	}
}

func TestTxnCommit(t *testing.T) {
	cli, srv := newTxnClient(t)
	srv.Handle("transact", func(params []json.RawMessage) (interface{}, error) {
		return []OperationResult{
			{Uuid: "0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"},
			{Count: 1},
		}, nil
	})

	txn := cli.Txn(testDb)
	txn.Insert(&ovn_nb.LogicalSwitch{Name: "ls0"})
	txn.Comment("claim vpc %s", "vpc0")
	results, err := txn.Commit(context.Background())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(results) != 2 || results[0].Uuid != "0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c" {
		t.Errorf("got %#v", results)
	}

	req := srv.LastRequest("transact")
	if req == nil || len(req.Params) != 3 {
		t.Fatalf("want a db and two operations, got %#v", req)
	}
	want := `{"op":"comment","comment":"claim vpc vpc0"}`
	if got := string(req.Params[2]); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestTxnCommitOperationError(t *testing.T) {
	cli, srv := newTxnClient(t)
	srv.Handle("transact", func(params []json.RawMessage) (interface{}, error) {
		return []OperationResult{
			{Error: "constraint violation", Details: "duplicate name"},
		}, nil
	})

	txn := cli.Txn(testDb)
	txn.Insert(&ovn_nb.LogicalSwitch{Name: "ls0"})
	results, err := txn.Commit(context.Background())
	if err == nil {
		t.Fatalf("want an error")
	}
	if errors.Cause(err) != ErrOperation {
		t.Errorf("want ErrOperation, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("the results should be handed back along with the error, got %#v", results)
	}
}

func TestTxnCommitEmpty(t *testing.T) {
	cli, srv := newTxnClient(t)
	results, err := cli.Txn(testDb).Commit(context.Background())
	if err != nil || results != nil {
		t.Errorf("got %#v, %v", results, err)
	}
	if req := srv.LastRequest("transact"); req != nil {
		t.Errorf("an empty transaction should not be sent")
	}
}
