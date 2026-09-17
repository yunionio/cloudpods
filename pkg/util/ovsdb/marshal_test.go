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
	"reflect"
	"strings"
	"testing"

	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/ovsdb/types"
)

func mustSchema(t *testing.T) *types.Schema {
	t.Helper()
	schema, err := types.ParseSchema(strings.NewReader(testSchema))
	if err != nil {
		t.Fatalf("parse test schema: %v", err)
	}
	return schema
}

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}

func TestRowFromIRow(t *testing.T) {
	schema := mustSchema(t)

	t.Run("zero columns are left out", func(t *testing.T) {
		lsp := &ovn_nb.LogicalSwitchPort{
			Name:      "lsp0",
			Addresses: []string{"00:11:22:33:44:55 10.0.0.2"},
			Options:   map[string]string{"router-port": "rp0"},
		}
		row, err := RowFromIRow(schema, lsp, false)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		want := `{"addresses":["set",["00:11:22:33:44:55 10.0.0.2"]],"name":"lsp0","options":["map",[["router-port","rp0"]]]}`
		if got := mustMarshal(t, row); got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	})

	t.Run("zero columns are included on request", func(t *testing.T) {
		lsp := &ovn_nb.LogicalSwitchPort{Name: "lsp0"}
		row, err := RowFromIRow(schema, lsp, true)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		// a full row update resets the columns that were left alone.  An
		// absent optional is the empty set, rfc7047 5.1
		want := `{"addresses":["set",[]],"dhcpv4_options":["set",[]],"enabled":["set",[]],` +
			`"external_ids":["map",[]],"name":"lsp0","options":["map",[]],"parent_name":["set",[]],` +
			`"port_security":["set",[]],"tag":["set",[]],"type":"","up":["set",[]]}`
		if got := mustMarshal(t, row); got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	})

	t.Run("optional scalars", func(t *testing.T) {
		var (
			tag    = int64(7)
			up     = true
			parent = "lsp-parent"
			dhcp   = "6bf5f9be-4c09-4c9e-8e1e-6d17b1b2e5f6"
		)
		lsp := &ovn_nb.LogicalSwitchPort{
			Name:          "lsp0",
			Tag:           &tag,
			Up:            &up,
			ParentName:    &parent,
			Dhcpv4Options: &dhcp,
		}
		row, err := RowFromIRow(schema, lsp, false)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		want := `{"dhcpv4_options":["uuid","6bf5f9be-4c09-4c9e-8e1e-6d17b1b2e5f6"],` +
			`"name":"lsp0","parent_name":"lsp-parent","tag":7,"up":true}`
		if got := mustMarshal(t, row); got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	})

	t.Run("uuid references may name a row of the transaction", func(t *testing.T) {
		dhcp := "@row1"
		lsp := &ovn_nb.LogicalSwitchPort{Name: "lsp0", Dhcpv4Options: &dhcp}
		row, err := RowFromIRow(schema, lsp, false)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		want := `{"dhcpv4_options":["named-uuid","row1"],"name":"lsp0"}`
		if got := mustMarshal(t, row); got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	})

	t.Run("uuid and version are never sent", func(t *testing.T) {
		lsp := &ovn_nb.LogicalSwitchPort{
			Uuid:    "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c",
			Version: "17c0b6a1-2b6d-4b03-8a1b-6f7c9b2e0a3d",
			Name:    "lsp0",
		}
		row, err := RowFromIRow(schema, lsp, true)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		if _, ok := row[ColUuid]; ok {
			t.Errorf("_uuid should not be part of the row")
		}
		if _, ok := row[ColVersion]; ok {
			t.Errorf("_version should not be part of the row")
		}
	})

	t.Run("columns the remote does not know are left out", func(t *testing.T) {
		// the test schema has no Logical_Switch_Port.ha_chassis_group, as
		// happens when the remote runs an older schema than the one the
		// row types were generated from
		hacg := "0dc7b2a6-6f7e-4a56-9b1a-2a9f6d6d0d51"
		lsp := &ovn_nb.LogicalSwitchPort{Name: "lsp0", HaChassisGroup: &hacg}
		row, err := RowFromIRow(schema, lsp, false)
		if err != nil {
			t.Fatalf("marshal row: %v", err)
		}
		if _, ok := row["ha_chassis_group"]; ok {
			t.Errorf("unknown column should have been left out: %#v", row)
		}
	})

	t.Run("unknown table", func(t *testing.T) {
		if _, err := RowFromIRow(schema, &ovn_nb.NAT{}, false); err == nil {
			t.Errorf("want an error for a table outside the schema")
		}
	})
}

func TestSetIRowFromRow(t *testing.T) {
	row := Row{
		"_uuid":        RawUuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"),
		"name":         "lsp0",
		"type":         "router",
		"addresses":    []interface{}{"set", []interface{}{"router"}},
		"tag":          []interface{}{"set", []interface{}{}},
		"up":           true,
		"options":      []interface{}{"map", []interface{}{[]interface{}{"router-port", "rp0"}}},
		"external_ids": []interface{}{"map", []interface{}{[]interface{}{"oc-ref", "net0"}}},
		// a column the generated type has no field for
		"no_such_column": "ignored",
	}
	lsp := &ovn_nb.LogicalSwitchPort{}
	if err := SetIRowFromRow(lsp, row); err != nil {
		t.Fatalf("unmarshal row: %v", err)
	}
	if lsp.Uuid != "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c" {
		t.Errorf("uuid: got %q", lsp.Uuid)
	}
	if lsp.Name != "lsp0" || lsp.Type != "router" {
		t.Errorf("name/type: got %q/%q", lsp.Name, lsp.Type)
	}
	if !reflect.DeepEqual(lsp.Addresses, []string{"router"}) {
		t.Errorf("addresses: got %#v", lsp.Addresses)
	}
	if lsp.Tag != nil {
		t.Errorf("tag: the empty set stands for an absent value, got %#v", lsp.Tag)
	}
	if lsp.Up == nil || !*lsp.Up {
		t.Errorf("up: got %#v", lsp.Up)
	}
	if !reflect.DeepEqual(lsp.Options, map[string]string{"router-port": "rp0"}) {
		t.Errorf("options: got %#v", lsp.Options)
	}
	if v, _ := lsp.GetExternalId("oc-ref"); v != "net0" {
		t.Errorf("external_ids: got %q", v)
	}
}

func TestRowRoundTrip(t *testing.T) {
	schema := mustSchema(t)
	var (
		tag    = int64(11)
		parent = "lsp-parent"
	)
	want := &ovn_nb.LogicalSwitchPort{
		Name:         "lsp0",
		Type:         "",
		Addresses:    []string{"00:11:22:33:44:55 10.0.0.2", "unknown"},
		PortSecurity: []string{"00:11:22:33:44:55"},
		Tag:          &tag,
		ParentName:   &parent,
		Options:      map[string]string{"a": "0", "b": "1"},
		ExternalIds:  map[string]string{"oc-ref": "gn0", "oc-version": "1"},
	}
	row, err := RowFromIRow(schema, want, false)
	if err != nil {
		t.Fatalf("marshal row: %v", err)
	}

	// go through the wire so that the values are the ones a remote sends
	// back, that is float64 numbers and []interface{} sets
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	wireRow := Row{}
	if err := json.Unmarshal(data, &wireRow); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	got := &ovn_nb.LogicalSwitchPort{}
	if err := SetIRowFromRow(got, wireRow); err != nil {
		t.Fatalf("unmarshal row: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}

func TestFillITable(t *testing.T) {
	tbl := &ovn_nb.LogicalSwitchPortTable{
		{Name: "stale"},
	}
	rows := []Row{
		{"_uuid": RawUuid("0a3b1a6e-1d2c-4f5b-9a8c-7d6e5f4a3b2c"), "name": "lsp0"},
		{"_uuid": RawUuid("1b4c2b7f-2e3d-5a6c-8b9d-6e5f4a3b2c1d"), "name": "lsp1"},
	}
	if err := FillITable(tbl, rows); err != nil {
		t.Fatalf("fill: %v", err)
	}
	if len(*tbl) != 2 {
		t.Fatalf("want the table to be replaced, got %d rows", len(*tbl))
	}
	if (*tbl)[0].Name != "lsp0" || (*tbl)[1].Name != "lsp1" {
		t.Errorf("got %#v", *tbl)
	}
}

func TestColumnsOf(t *testing.T) {
	schema := mustSchema(t)
	cols, err := ColumnsOf(schema, "DHCP_Options")
	if err != nil {
		t.Fatalf("columns: %v", err)
	}
	want := []string{"_uuid", "_version", "cidr", "external_ids", "options"}
	if !reflect.DeepEqual(cols, want) {
		t.Errorf("got %#v, want %#v", cols, want)
	}
	if _, err := ColumnsOf(schema, "No_Such_Table"); err == nil {
		t.Errorf("want an error for an unknown table")
	}
}
