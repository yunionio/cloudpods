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

	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

// Select reads the rows of a table matching the conditions.  Passing no
// columns asks for all of them
func (cli *Client) Select(ctx context.Context, db, table string, where Conditions, columns ...string) ([]Row, error) {
	ops := []Operation{SelectOp(table, where, columns...)}
	results, err := cli.Transact(ctx, db, ops)
	if err != nil {
		return nil, err
	}
	if err := CheckOperationResults(results, ops); err != nil {
		return nil, err
	}
	return results[0].Rows, nil
}

// DumpTables reads whole tables into the generated table types, replacing
// whatever they held.  It is the rfc7047 counterpart of running
// "ovn-nbctl --format=json list <table>" once per table, except that it takes
// a single transaction and a single round trip
//
//	db := ovn_nb.OVNNorthbound{}
//	err := cli.DumpTables(ctx, "OVN_Northbound", &db.LogicalSwitch, &db.LogicalSwitchPort)
func (cli *Client) DumpTables(ctx context.Context, db string, itbls ...types.ITable) error {
	if len(itbls) == 0 {
		return nil
	}
	if _, err := cli.schemaOf(db); err != nil {
		return err
	}
	ops := make([]Operation, len(itbls))
	for i, itbl := range itbls {
		ops[i] = SelectOp(itbl.OvsdbTableName(), Where())
	}
	results, err := cli.Transact(ctx, db, ops)
	if err != nil {
		return err
	}
	if err := CheckOperationResults(results, ops); err != nil {
		return err
	}
	for i, itbl := range itbls {
		if err := FillITable(itbl, results[i].Rows); err != nil {
			return errors.Wrapf(err, "fill %s", itbl.OvsdbTableName())
		}
	}
	return nil
}
