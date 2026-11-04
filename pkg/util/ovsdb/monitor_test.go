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

const (
	uuidLs0  = "0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"
	uuidLsp0 = "1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"
	uuidLsp1 = "2c5d3c80-3f4e-6b7d-9cae-7f605b4c3d2e"
)

func newMonitorClient(t *testing.T) (*Client, *fakeServer) {
	t.Helper()
	cli, srv := newFakeServer(t, nil)
	if _, err := cli.GetSchema(context.Background(), testDb); err != nil {
		t.Fatalf("get_schema: %v", err)
	}
	srv.Handle("monitor", func(params []json.RawMessage) (interface{}, error) {
		return TableUpdates{
			"Logical_Switch": TableUpdate{
				uuidLs0: &RowUpdate{New: Row{"name": "ls0"}},
			},
			"Logical_Switch_Port": TableUpdate{
				uuidLsp0: &RowUpdate{New: Row{"name": "lsp0", "type": ""}},
				uuidLsp1: &RowUpdate{New: Row{"name": "lsp1", "type": "router"}},
			},
		}, nil
	})
	srv.Handle("monitor_cancel", func(params []json.RawMessage) (interface{}, error) {
		return map[string]interface{}{}, nil
	})
	return cli, srv
}

// sync waits for the notifications sent so far to have been processed.  The
// read loop handles messages one after another, so a round trip of our own is
// enough to know the earlier ones are done with
func syncClient(t *testing.T, cli *Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Echo(ctx); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

func TestMonitorInitial(t *testing.T) {
	cli, srv := newMonitorClient(t)

	mon, err := cli.MonitorTables(context.Background(), testDb,
		&ovn_nb.LogicalSwitchTable{}, &ovn_nb.LogicalSwitchPortTable{})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	// the rows the remote sent along with the reply are cached already
	if got := mon.Cache().Len("Logical_Switch_Port"); got != 2 {
		t.Errorf("want 2 ports cached, got %d", got)
	}
	row, ok := mon.Cache().Row("Logical_Switch", uuidLs0)
	if !ok {
		t.Fatalf("the switch should be cached")
	}
	if row["name"] != "ls0" {
		t.Errorf("name: got %#v", row["name"])
	}
	// the uuid is the key of the update, the cache fills it in as a column
	if row.Uuid() != uuidLs0 {
		t.Errorf("uuid: got %q", row.Uuid())
	}

	req := srv.LastRequest("monitor")
	if req == nil || len(req.Params) != 3 {
		t.Fatalf("got %#v", req)
	}
	var reqs MonitorRequests
	if err := json.Unmarshal(req.Params[2], &reqs); err != nil {
		t.Fatalf("monitor requests: %v", err)
	}
	if len(reqs) != 2 {
		t.Errorf("want 2 tables monitored, got %#v", reqs)
	}
}

func TestMonitorUpdates(t *testing.T) {
	cli, srv := newMonitorClient(t)
	mon, err := cli.MonitorTables(context.Background(), testDb, &ovn_nb.LogicalSwitchPortTable{})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	updates := make(chan TableUpdates, 4)
	mon.OnUpdate(func(tus TableUpdates) {
		updates <- tus
	})

	// rfc7047 4.1.6, a modify carries only the columns that changed
	err = srv.Notify("update", mon.Id(), TableUpdates{
		"Logical_Switch_Port": TableUpdate{
			uuidLsp0: &RowUpdate{
				New: Row{"type": "localnet"},
				Old: Row{"type": ""},
			},
		},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	select {
	case <-updates:
	case <-time.After(5 * time.Second):
		t.Fatalf("the update handler was never called")
	}

	row, ok := mon.Cache().Row("Logical_Switch_Port", uuidLsp0)
	if !ok {
		t.Fatalf("the port should still be cached")
	}
	if row["type"] != "localnet" {
		t.Errorf("type: got %#v", row["type"])
	}
	if row["name"] != "lsp0" {
		t.Errorf("a modify should leave the other columns alone, got %#v", row)
	}

	// an insert
	err = srv.Notify("update", mon.Id(), TableUpdates{
		"Logical_Switch_Port": TableUpdate{
			uuidLs0: &RowUpdate{New: Row{"name": "lsp2"}},
		},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	<-updates
	if got := mon.Cache().Len("Logical_Switch_Port"); got != 3 {
		t.Errorf("want 3 ports cached, got %d", got)
	}

	// a delete, which comes with old only
	err = srv.Notify("update", mon.Id(), TableUpdates{
		"Logical_Switch_Port": TableUpdate{
			uuidLsp1: &RowUpdate{Old: Row{"name": "lsp1"}},
		},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	<-updates
	if _, ok := mon.Cache().Row("Logical_Switch_Port", uuidLsp1); ok {
		t.Errorf("the port should be gone from the cache")
	}
}

func TestMonitorFillTables(t *testing.T) {
	cli, _ := newMonitorClient(t)
	mon, err := cli.MonitorTables(context.Background(), testDb,
		&ovn_nb.LogicalSwitchTable{}, &ovn_nb.LogicalSwitchPortTable{})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	db := ovn_nb.OVNNorthbound{}
	if err := mon.Cache().FillTables(&db.LogicalSwitch, &db.LogicalSwitchPort); err != nil {
		t.Fatalf("fill: %v", err)
	}
	if len(db.LogicalSwitch) != 1 || db.LogicalSwitch[0].Name != "ls0" {
		t.Errorf("switches: got %#v", db.LogicalSwitch)
	}
	if db.LogicalSwitch[0].Uuid != uuidLs0 {
		t.Errorf("uuid: got %q", db.LogicalSwitch[0].Uuid)
	}
	// rows come out ordered by uuid
	if len(db.LogicalSwitchPort) != 2 ||
		db.LogicalSwitchPort[0].Name != "lsp0" ||
		db.LogicalSwitchPort[1].Name != "lsp1" {
		t.Errorf("ports: got %#v", db.LogicalSwitchPort)
	}
	// filling again replaces rather than appends
	if err := mon.Cache().FillTables(&db.LogicalSwitch); err != nil {
		t.Fatalf("fill: %v", err)
	}
	if len(db.LogicalSwitch) != 1 {
		t.Errorf("want the table replaced, got %#v", db.LogicalSwitch)
	}
}

func TestMonitorCancel(t *testing.T) {
	cli, srv := newMonitorClient(t)
	mon, err := cli.MonitorTables(context.Background(), testDb, &ovn_nb.LogicalSwitchPortTable{})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if err := mon.Cancel(context.Background()); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if req := srv.LastRequest("monitor_cancel"); req == nil {
		t.Errorf("monitor_cancel was never sent")
	}
	// cancelling twice is not an error, and updates in flight are ignored
	if err := mon.Cancel(context.Background()); err != nil {
		t.Errorf("cancel again: %v", err)
	}
	if err := srv.Notify("update", mon.Id(), TableUpdates{
		"Logical_Switch_Port": TableUpdate{
			uuidLsp0: &RowUpdate{Old: Row{"name": "lsp0"}},
		},
	}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	syncClient(t, cli)
	if got := mon.Cache().Len("Logical_Switch_Port"); got != 2 {
		t.Errorf("updates of a cancelled monitor should be dropped, got %d rows", got)
	}
}

func TestMonitorUnknownTable(t *testing.T) {
	cli, _ := newMonitorClient(t)
	_, err := cli.Monitor(context.Background(), testDb, MonitorRequests{
		"No_Such_Table": MonitorRequest{},
	})
	if errors.Cause(err) != ErrSchema {
		t.Errorf("want ErrSchema, got %v", err)
	}
}

func TestMonitorNeedsSchema(t *testing.T) {
	cli, _ := newFakeServer(t, nil)
	if _, err := cli.MonitorAll(context.Background(), testDb); errors.Cause(err) != ErrSchema {
		t.Errorf("want ErrSchema, got %v", err)
	}
}

func TestCacheApply(t *testing.T) {
	schema := mustSchema(t)
	cache := NewCache(schema)

	cache.Apply(TableUpdates{
		"Logical_Switch": TableUpdate{
			uuidLs0: &RowUpdate{New: Row{"name": "ls0", "other_config": Map{}}},
		},
	})
	if !reflect.DeepEqual(cache.Tables(), []string{"Logical_Switch"}) {
		t.Errorf("tables: got %#v", cache.Tables())
	}

	// a row that is modified and then deleted leaves the table empty, and
	// an empty table is forgotten about
	cache.Apply(TableUpdates{
		"Logical_Switch": TableUpdate{
			uuidLs0: &RowUpdate{Old: Row{"name": "ls0"}},
		},
	})
	if got := cache.Len("Logical_Switch"); got != 0 {
		t.Errorf("want the row gone, got %d", got)
	}
	if len(cache.Tables()) != 0 {
		t.Errorf("tables: got %#v", cache.Tables())
	}

	// deleting a row that was never there is not an error
	cache.Apply(TableUpdates{
		"Logical_Switch": TableUpdate{
			uuidLs0: &RowUpdate{Old: Row{"name": "ls0"}},
		},
	})

	// rows handed out are copies, changing them does not change the cache
	cache.Apply(TableUpdates{
		"Logical_Switch": TableUpdate{
			uuidLs0: &RowUpdate{New: Row{"name": "ls0"}},
		},
	})
	row, _ := cache.Row("Logical_Switch", uuidLs0)
	row["name"] = "tampered"
	again, _ := cache.Row("Logical_Switch", uuidLs0)
	if again["name"] != "ls0" {
		t.Errorf("the cache should hand out copies, got %#v", again)
	}
}
