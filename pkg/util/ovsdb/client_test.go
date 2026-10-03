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
	"reflect"
	"testing"
	"time"

	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/pkg/errors"
)

const testDb = "OVN_Northbound"

func TestParseAddr(t *testing.T) {
	cases := []struct {
		in      string
		network string
		address string
		tls     bool
		err     bool
	}{
		{in: "unix:/var/run/ovn/ovnnb_db.sock", network: "unix", address: "/var/run/ovn/ovnnb_db.sock"},
		{in: "punix:/var/run/ovn/ovnnb_db.sock", network: "unix", address: "/var/run/ovn/ovnnb_db.sock"},
		{in: "tcp:127.0.0.1:6641", network: "tcp", address: "127.0.0.1:6641"},
		{in: "tcp:[::1]:6641", network: "tcp", address: "[::1]:6641"},
		{in: "ssl:127.0.0.1:6641", network: "tcp", address: "127.0.0.1:6641", tls: true},
		{in: "/var/run/ovn/ovnnb_db.sock", network: "unix", address: "/var/run/ovn/ovnnb_db.sock"},
		{in: "127.0.0.1:6641", network: "tcp", address: "127.0.0.1:6641"},
		{in: "nonsense", err: true},
	}
	for _, c := range cases {
		network, address, wantTLS, err := parseAddr(c.in)
		if c.err {
			if err == nil {
				t.Errorf("%q: want an error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if network != c.network || address != c.address || wantTLS != c.tls {
			t.Errorf("%q: got %s %s tls=%v", c.in, network, address, wantTLS)
		}
	}
}

func TestListDbs(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	srv.Handle("list_dbs", func(params []json.RawMessage) (interface{}, error) {
		return []string{"OVN_Northbound", "_Server"}, nil
	})
	dbs, err := cli.ListDbs(context.Background())
	if err != nil {
		t.Fatalf("list_dbs: %v", err)
	}
	if !reflect.DeepEqual(dbs, []string{"OVN_Northbound", "_Server"}) {
		t.Errorf("got %#v", dbs)
	}
}

func TestGetSchema(t *testing.T) {
	cli, _ := newFakeServer(t, nil)
	if cli.Schema(testDb) != nil {
		t.Errorf("no schema should be known before get_schema")
	}
	schema, err := cli.GetSchema(context.Background(), testDb)
	if err != nil {
		t.Fatalf("get_schema: %v", err)
	}
	if schema.Name != testDb {
		t.Errorf("name: got %q", schema.Name)
	}
	if _, ok := schema.Tables["Logical_Switch_Port"]; !ok {
		t.Errorf("tables: got %#v", schema.Tables)
	}
	if cli.Schema(testDb) != schema {
		t.Errorf("the schema should have been remembered")
	}
}

func TestRemoteError(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	srv.Handle("list_dbs", func(params []json.RawMessage) (interface{}, error) {
		return nil, errors.Error("no can do")
	})
	_, err := cli.ListDbs(context.Background())
	if err == nil {
		t.Fatalf("want an error")
	}
	if errors.Cause(err) != ErrRemote {
		t.Errorf("want ErrRemote, got %v", err)
	}
}

func TestEchoFromRemote(t *testing.T) {
	// rfc7047 4.1.11, the remote probes us and we must answer with the very
	// same params
	cli, srv := newFakeServer(t, nil)
	if err := srv.Request(1000, "echo", "probe"); err != nil {
		t.Fatalf("echo request: %v", err)
	}
	reply := srv.WaitReply(t)
	if string(reply.Id) != "1000" {
		t.Errorf("id: got %s", string(reply.Id))
	}
	if got := string(reply.Result); got != `["probe"]` {
		t.Errorf("result: got %s, want the params back", got)
	}
	if !isNullJson(reply.Error) {
		t.Errorf("error: got %s", string(reply.Error))
	}
	// the connection must still work after having answered the probe
	if err := cli.Echo(context.Background()); err != nil {
		t.Fatalf("echo: %v", err)
	}
}

func TestCallOnClosedConnection(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	srv.Close()
	// give the read loop a chance to notice
	select {
	case <-cli.Done():
	case <-time.After(5 * time.Second):
		t.Fatalf("the client did not notice the connection going away")
	}
	if _, err := cli.ListDbs(context.Background()); err == nil {
		t.Errorf("want an error once the connection is gone")
	}
}

func TestCallContextCancel(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	block := make(chan struct{})
	srv.Handle("list_dbs", func(params []json.RawMessage) (interface{}, error) {
		<-block
		return []string{}, nil
	})
	defer close(block)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := cli.ListDbs(ctx); err == nil {
		t.Errorf("want an error when the context runs out")
	}
}

func TestSelect(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	if _, err := cli.GetSchema(context.Background(), testDb); err != nil {
		t.Fatalf("get_schema: %v", err)
	}
	srv.Handle("transact", func(params []json.RawMessage) (interface{}, error) {
		return []OperationResult{
			{Rows: []Row{{"name": "ls0"}}},
		}, nil
	})
	rows, err := cli.Select(context.Background(), testDb, "Logical_Switch", Where())
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 || rows[0]["name"] != "ls0" {
		t.Errorf("got %#v", rows)
	}

	req := srv.LastRequest("transact")
	if req == nil || len(req.Params) != 2 {
		t.Fatalf("got %#v", req)
	}
	var db string
	if err := json.Unmarshal(req.Params[0], &db); err != nil || db != testDb {
		t.Errorf("db: got %q, %v", db, err)
	}
	want := `{"op":"select","table":"Logical_Switch","where":[]}`
	if got := string(req.Params[1]); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestDumpTables(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	if _, err := cli.GetSchema(context.Background(), testDb); err != nil {
		t.Fatalf("get_schema: %v", err)
	}
	srv.Handle("transact", func(params []json.RawMessage) (interface{}, error) {
		return []OperationResult{
			{Rows: []Row{
				{"_uuid": RawUuid("0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"), "name": "ls0"},
			}},
			{Rows: []Row{
				{"_uuid": RawUuid("1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"), "name": "lsp0"},
				{"_uuid": RawUuid("2c5d3c80-3f4e-6b7d-9cae-7f605b4c3d2e"), "name": "lsp1"},
			}},
		}, nil
	})

	db := ovn_nb.OVNNorthbound{}
	err := cli.DumpTables(context.Background(), testDb, &db.LogicalSwitch, &db.LogicalSwitchPort)
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	if len(db.LogicalSwitch) != 1 || db.LogicalSwitch[0].Name != "ls0" {
		t.Errorf("logical switch: got %#v", db.LogicalSwitch)
	}
	if len(db.LogicalSwitchPort) != 2 || db.LogicalSwitchPort[1].Name != "lsp1" {
		t.Errorf("logical switch port: got %#v", db.LogicalSwitchPort)
	}

	req := srv.LastRequest("transact")
	if req == nil || len(req.Params) != 3 {
		t.Fatalf("a single transaction with two selects was expected, got %#v", req)
	}
}

func TestDumpTablesNeedsSchema(t *testing.T) {
	cli, _ := newFakeServer(t, nil)
	db := ovn_nb.OVNNorthbound{}
	err := cli.DumpTables(context.Background(), testDb, &db.LogicalSwitch)
	if err == nil {
		t.Fatalf("want an error when the schema was never fetched")
	}
	if errors.Cause(err) != ErrSchema {
		t.Errorf("want ErrSchema, got %v", err)
	}
}

func TestLock(t *testing.T) {
	cli, srv := newFakeServer(t, nil)
	srv.Handle("lock", func(params []json.RawMessage) (interface{}, error) {
		return map[string]bool{"locked": false}, nil
	})
	srv.Handle("unlock", func(params []json.RawMessage) (interface{}, error) {
		return map[string]interface{}{}, nil
	})

	locked, err := cli.Lock(context.Background(), "ovn_nb_lock")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if locked || cli.Locked("ovn_nb_lock") {
		t.Errorf("the lock should not be held yet")
	}

	// rfc7047 4.1.9, the remote tells us when the lock becomes ours
	if err := srv.Notify("locked", "ovn_nb_lock"); err != nil {
		t.Fatalf("locked notification: %v", err)
	}
	// the notification is processed by the read loop, sync with it by
	// making a call that has to go through the same connection
	if err := cli.Echo(context.Background()); err != nil {
		t.Fatalf("echo: %v", err)
	}
	if !cli.Locked("ovn_nb_lock") {
		t.Errorf("the lock should be held now")
	}

	if err := srv.Notify("stolen", "ovn_nb_lock"); err != nil {
		t.Fatalf("stolen notification: %v", err)
	}
	if err := cli.Echo(context.Background()); err != nil {
		t.Fatalf("echo: %v", err)
	}
	if cli.Locked("ovn_nb_lock") {
		t.Errorf("the lock was stolen, it should not be held")
	}

	if err := cli.Unlock(context.Background(), "ovn_nb_lock"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
}
