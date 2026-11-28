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

// Package ovsdb implements the Open vSwitch Database Management Protocol as
// specified in RFC7047.
//
// It speaks JSON-RPC 1.0 over a unix socket, a plain tcp connection, or a tls
// connection, and implements the client side of the following methods
//
//	list_dbs, get_schema, transact, cancel, monitor, update, monitor_cancel,
//	lock, steal, unlock, locked, stolen, echo
//
// Rows are carried in the value format of RFC7047 5.1, and are converted
// to and from the generated Go structs of yunion.io/x/ovsdb with the help of
// the database schema retrieved at runtime with get_schema.  That is,
// callers keep using types like ovn_nb.LogicalSwitchPort and never have to
// deal with ["set", [...]] and friends themselves.
//
// A typical read-write session looks like
//
//	cli, err := ovsdb.Dial(ctx, "unix:/var/run/ovn/ovnnb_db.sock", nil)
//	defer cli.Close()
//	if _, err := cli.GetSchema(ctx, "OVN_Northbound"); err != nil { ... }
//
//	txn := cli.Txn("OVN_Northbound")
//	lsRef := txn.Insert(&ovn_nb.LogicalSwitch{Name: "ls0"})
//	lspRef := txn.Insert(&ovn_nb.LogicalSwitchPort{Name: "lsp0"})
//	txn.AddRefWhere("Logical_Switch", ovsdb.WhereUuidRef(lsRef), "ports", lspRef)
//	if _, err := txn.Commit(ctx); err != nil { ... }
//
// Keeping a local copy of a database in sync is done with Monitor
//
//	mon, err := cli.MonitorAll(ctx, "OVN_Northbound")
//	db := ovn_nb.OVNNorthbound{}
//	err = mon.Cache().FillTables(&db.LogicalSwitch, &db.LogicalSwitchPort)
package ovsdb // import "yunion.io/x/onecloud/pkg/util/ovsdb"
