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
	"fmt"

	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

// Txn collects operations to be sent to the remote as a single transaction.
//
// Errors met while building the transaction are remembered and reported by
// Commit, so a chain of calls can be written without checking each one of
// them.  Once an error has been recorded no further operation is added
type Txn struct {
	cli    *Client
	db     string
	schema *types.Schema

	ops []Operation
	err error

	nUuid int
}

// Txn starts a transaction against a database.  The schema of the database
// must have been fetched with GetSchema beforehand
func (cli *Client) Txn(db string) *Txn {
	txn := &Txn{
		cli: cli,
		db:  db,
	}
	schema, err := cli.schemaOf(db)
	if err != nil {
		txn.err = err
		return txn
	}
	txn.schema = schema
	return txn
}

// Err is the first error met while building the transaction
func (txn *Txn) Err() error { return txn.err }

// Ops are the operations collected so far
func (txn *Txn) Ops() []Operation { return txn.ops }

// Len is the number of operations collected so far
func (txn *Txn) Len() int { return len(txn.ops) }

func (txn *Txn) add(op Operation) *Txn {
	if txn.err != nil {
		return txn
	}
	txn.ops = append(txn.ops, op)
	return txn
}

func (txn *Txn) fail(err error) *Txn {
	if txn.err == nil {
		txn.err = err
	}
	return txn
}

// newUuidName makes up a name for a row inserted by this transaction.  rfc7047
// 5.1 wants <id>s, which start with a letter or an underscore
func (txn *Txn) newUuidName() string {
	txn.nUuid++
	return fmt.Sprintf("row%d", txn.nUuid)
}

// Insert adds an insert operation for the row and returns the reference other
// operations of the same transaction can use to point at it.
//
// Rows of tables that are not at the root of the database are garbage
// collected right away unless something refers to them, so the reference
// normally goes into a column of a parent row, see AddRef
func (txn *Txn) Insert(irow types.IRow) string {
	return txn.InsertAs(txn.newUuidName(), irow)
}

// InsertAs is Insert with a caller chosen name
func (txn *Txn) InsertAs(uuidName string, irow types.IRow) string {
	if txn.err != nil {
		return NamedUuidPrefix + uuidName
	}
	row, err := RowFromIRow(txn.schema, irow, false)
	if err != nil {
		txn.fail(err)
		return NamedUuidPrefix + uuidName
	}
	txn.add(InsertOp(irow.OvsdbTableName(), row, uuidName))
	return NamedUuidPrefix + uuidName
}

// Update replaces the writable columns of the row identified by its uuid.
// Columns left at their zero value on the Go side are reset to the schema
// default, which is what a full row update means
func (txn *Txn) Update(irow types.IRow) *Txn {
	uuid := irow.OvsdbUuid()
	if uuid == "" {
		return txn.fail(errors.Wrapf(ErrValue, "%s: update needs a uuid", irow.OvsdbTableName()))
	}
	if txn.err != nil {
		return txn
	}
	row, err := RowFromIRow(txn.schema, irow, true)
	if err != nil {
		return txn.fail(err)
	}
	return txn.add(UpdateOp(irow.OvsdbTableName(), WhereUuid(uuid), row))
}

// UpdateColumns updates only the columns named, leaving the others alone
func (txn *Txn) UpdateColumns(irow types.IRow, columns ...string) *Txn {
	uuid := irow.OvsdbUuid()
	if uuid == "" {
		return txn.fail(errors.Wrapf(ErrValue, "%s: update needs a uuid", irow.OvsdbTableName()))
	}
	if txn.err != nil {
		return txn
	}
	full, err := RowFromIRow(txn.schema, irow, true)
	if err != nil {
		return txn.fail(err)
	}
	row := Row{}
	for _, col := range columns {
		val, ok := full[col]
		if !ok {
			return txn.fail(errors.Wrapf(ErrSchema, "%s: no such writable column %s",
				irow.OvsdbTableName(), col))
		}
		row[col] = val
	}
	return txn.add(UpdateOp(irow.OvsdbTableName(), WhereUuid(uuid), row))
}

// UpdateWhere updates the rows matching the conditions with a raw row
func (txn *Txn) UpdateWhere(table string, where Conditions, row Row) *Txn {
	return txn.add(UpdateOp(table, where, row))
}

// Delete removes the row identified by its uuid
func (txn *Txn) Delete(irow types.IRow) *Txn {
	uuid := irow.OvsdbUuid()
	if uuid == "" {
		return txn.fail(errors.Wrapf(ErrValue, "%s: delete needs a uuid", irow.OvsdbTableName()))
	}
	return txn.add(DeleteOp(irow.OvsdbTableName(), WhereUuid(uuid)))
}

// DeleteWhere removes the rows matching the conditions
func (txn *Txn) DeleteWhere(table string, where Conditions) *Txn {
	return txn.add(DeleteOp(table, where))
}

// Select reads rows.  Passing no columns asks for all of them
func (txn *Txn) Select(table string, where Conditions, columns ...string) *Txn {
	return txn.add(SelectOp(table, where, columns...))
}

// Mutate applies mutations to the rows matching the conditions
func (txn *Txn) Mutate(table string, where Conditions, mutations ...Mutation) *Txn {
	return txn.add(MutateOp(table, where, mutations...))
}

// AddRef inserts references into a set column of a parent row.  It is the way
// a row inserted by Insert is hooked into the database
//
//	lspRef := txn.Insert(&ovn_nb.LogicalSwitchPort{Name: "lsp0"})
//	txn.AddRef(ls, "ports", lspRef)
func (txn *Txn) AddRef(parent types.IRow, column string, refs ...string) *Txn {
	uuid := parent.OvsdbUuid()
	if uuid == "" {
		return txn.fail(errors.Wrapf(ErrValue, "%s: add ref needs a uuid", parent.OvsdbTableName()))
	}
	return txn.AddRefWhere(parent.OvsdbTableName(), WhereUuid(uuid), column, refs...)
}

// AddRefWhere is AddRef for a parent identified by conditions, which is what
// a parent inserted by the same transaction needs
func (txn *Txn) AddRefWhere(table string, where Conditions, column string, refs ...string) *Txn {
	if len(refs) == 0 {
		return txn
	}
	return txn.add(MutateOp(table, where, Mutation{
		Column:  column,
		Mutator: MutInsert,
		Value:   Set(UuidValues(refs)),
	}))
}

// DelRef removes references from a set column of a parent row.  Removing the
// last reference to a row of a non root table has the remote delete that row
func (txn *Txn) DelRef(parent types.IRow, column string, refs ...string) *Txn {
	uuid := parent.OvsdbUuid()
	if uuid == "" {
		return txn.fail(errors.Wrapf(ErrValue, "%s: del ref needs a uuid", parent.OvsdbTableName()))
	}
	return txn.DelRefWhere(parent.OvsdbTableName(), WhereUuid(uuid), column, refs...)
}

// DelRefWhere is DelRef for a parent identified by conditions
func (txn *Txn) DelRefWhere(table string, where Conditions, column string, refs ...string) *Txn {
	if len(refs) == 0 {
		return txn
	}
	return txn.add(MutateOp(table, where, Mutation{
		Column:  column,
		Mutator: MutDelete,
		Value:   Set(UuidValues(refs)),
	}))
}

// Wait makes the transaction fail unless the rows selected have the expected
// contents.  It is how a read modify write cycle is made safe against
// concurrent writers, rfc7047 5.2.6
func (txn *Txn) Wait(table string, where Conditions, columns []string, until string, rows []Row, timeout *int) *Txn {
	return txn.add(WaitOp(table, where, columns, until, rows, timeout))
}

// Comment attaches a comment to the transaction.  ovsdb-server writes it to
// its log along with the transaction
func (txn *Txn) Comment(format string, args ...interface{}) *Txn {
	return txn.add(CommentOp(fmt.Sprintf(format, args...)))
}

// Assert makes the transaction fail unless the lock is held
func (txn *Txn) Assert(lock string) *Txn {
	return txn.add(AssertOp(lock))
}

// Commit sends the transaction and checks the results.  A transaction
// without operations is not sent at all
func (txn *Txn) Commit(ctx context.Context) ([]OperationResult, error) {
	if txn.err != nil {
		return nil, txn.err
	}
	if len(txn.ops) == 0 {
		return nil, nil
	}
	results, err := txn.cli.Transact(ctx, txn.db, txn.ops)
	if err != nil {
		return nil, err
	}
	if err := CheckOperationResults(results, txn.ops); err != nil {
		return results, err
	}
	return results, nil
}
