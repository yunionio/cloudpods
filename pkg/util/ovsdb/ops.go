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

	"yunion.io/x/pkg/errors"
)

// The operations of rfc7047 5.2
const (
	OpInsert  = "insert"
	OpSelect  = "select"
	OpUpdate  = "update"
	OpMutate  = "mutate"
	OpDelete  = "delete"
	OpWait    = "wait"
	OpCommit  = "commit"
	OpAbort   = "abort"
	OpComment = "comment"
	OpAssert  = "assert"
)

// The condition functions of rfc7047 5.1
const (
	FnLt       = "<"
	FnLe       = "<="
	FnEq       = "=="
	FnNe       = "!="
	FnGe       = ">="
	FnGt       = ">"
	FnIncludes = "includes"
	FnExcludes = "excludes"
)

// The mutators of rfc7047 5.1
const (
	MutSum        = "+="
	MutDifference = "-="
	MutProduct    = "*="
	MutQuotient   = "/="
	MutRemain     = "%="
	MutInsert     = "insert"
	MutDelete     = "delete"
)

// Condition is a rfc7047 <condition>, a 3 element array
type Condition struct {
	Column   string
	Function string
	Value    interface{}
}

func (c Condition) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]interface{}{c.Column, c.Function, c.Value})
}

func (c *Condition) UnmarshalJSON(data []byte) error {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) != 3 {
		return errors.Wrapf(ErrValue, "condition: want 3 elements, got %d", len(arr))
	}
	col, ok := arr[0].(string)
	if !ok {
		return errors.Wrapf(ErrValue, "condition: column not a string: %#v", arr[0])
	}
	fn, ok := arr[1].(string)
	if !ok {
		return errors.Wrapf(ErrValue, "condition: function not a string: %#v", arr[1])
	}
	c.Column, c.Function, c.Value = col, fn, arr[2]
	return nil
}

// Conditions is the "where" member of an operation.  The empty, but non nil,
// value matches every row of the table
type Conditions []Condition

// Where builds a set of conditions
func Where(conds ...Condition) Conditions {
	if conds == nil {
		conds = Conditions{}
	}
	return conds
}

// WhereUuid matches the single row with the given uuid
func WhereUuid(uuid string) Conditions {
	return Conditions{
		{Column: "_uuid", Function: FnEq, Value: Uuid(uuid)},
	}
}

// WhereUuidRef matches a single row by uuid or by a named uuid of the
// ongoing transaction
func WhereUuidRef(ref string) Conditions {
	return Conditions{
		{Column: "_uuid", Function: FnEq, Value: UuidValue(ref)},
	}
}

// Mutation is a rfc7047 <mutation>, a 3 element array
type Mutation struct {
	Column  string
	Mutator string
	Value   interface{}
}

func (m Mutation) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]interface{}{m.Column, m.Mutator, m.Value})
}

func (m *Mutation) UnmarshalJSON(data []byte) error {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) != 3 {
		return errors.Wrapf(ErrValue, "mutation: want 3 elements, got %d", len(arr))
	}
	col, ok := arr[0].(string)
	if !ok {
		return errors.Wrapf(ErrValue, "mutation: column not a string: %#v", arr[0])
	}
	mut, ok := arr[1].(string)
	if !ok {
		return errors.Wrapf(ErrValue, "mutation: mutator not a string: %#v", arr[1])
	}
	m.Column, m.Mutator, m.Value = col, mut, arr[2]
	return nil
}

// Operation is one element of the params of a transact request.
//
// Members that are optional for some operations and mandatory for others are
// held by pointer so that the difference between "absent" and "empty" is
// kept.  A select with Where set to an empty, non nil, Conditions selects the
// whole table, a select with Where left nil is a malformed request
type Operation struct {
	Op        string      `json:"op"`
	Table     string      `json:"table,omitempty"`
	Row       Row         `json:"row,omitempty"`
	Rows      []Row       `json:"rows,omitempty"`
	Columns   []string    `json:"columns,omitempty"`
	Mutations []Mutation  `json:"mutations,omitempty"`
	Timeout   *int        `json:"timeout,omitempty"`
	Where     *Conditions `json:"where,omitempty"`
	Until     string      `json:"until,omitempty"`
	Durable   *bool       `json:"durable,omitempty"`
	Comment   string      `json:"comment,omitempty"`
	Lock      string      `json:"lock,omitempty"`
	UuidName  string      `json:"uuid-name,omitempty"`
}

// OperationResult is one element of the result of a transact request
type OperationResult struct {
	Count   int    `json:"count"`
	Error   string `json:"error"`
	Details string `json:"details"`
	Rows    []Row  `json:"rows"`
	// Uuid is the uuid of the row inserted by an insert operation
	Uuid resultUuid `json:"uuid"`
}

// resultUuid decodes the ["uuid", <s>] form results carry
type resultUuid string

func (u *resultUuid) UnmarshalJSON(data []byte) error {
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	if val == nil {
		*u = ""
		return nil
	}
	uuid, err := ParseUuid(val)
	if err != nil {
		return err
	}
	*u = resultUuid(uuid)
	return nil
}

func (u resultUuid) MarshalJSON() ([]byte, error) {
	if u == "" {
		return []byte("null"), nil
	}
	return json.Marshal(Uuid(u))
}

// SelectOp builds a select operation.  Passing no columns asks for all of them
func SelectOp(table string, where Conditions, columns ...string) Operation {
	where = Where(where...)
	return Operation{
		Op:      OpSelect,
		Table:   table,
		Where:   &where,
		Columns: columns,
	}
}

// InsertOp builds an insert operation.  uuidName may be empty when the new
// row does not need to be referred to by the rest of the transaction
func InsertOp(table string, row Row, uuidName string) Operation {
	return Operation{
		Op:       OpInsert,
		Table:    table,
		Row:      row,
		UuidName: uuidName,
	}
}

// UpdateOp builds an update operation
func UpdateOp(table string, where Conditions, row Row) Operation {
	where = Where(where...)
	return Operation{
		Op:    OpUpdate,
		Table: table,
		Where: &where,
		Row:   row,
	}
}

// MutateOp builds a mutate operation
func MutateOp(table string, where Conditions, mutations ...Mutation) Operation {
	where = Where(where...)
	return Operation{
		Op:        OpMutate,
		Table:     table,
		Where:     &where,
		Mutations: mutations,
	}
}

// DeleteOp builds a delete operation
func DeleteOp(table string, where Conditions) Operation {
	where = Where(where...)
	return Operation{
		Op:    OpDelete,
		Table: table,
		Where: &where,
	}
}

// WaitOp builds a wait operation.  timeout is in milliseconds, a nil timeout
// asks the remote to wait indefinitely
func WaitOp(table string, where Conditions, columns []string, until string, rows []Row, timeout *int) Operation {
	where = Where(where...)
	if rows == nil {
		rows = []Row{}
	}
	return Operation{
		Op:      OpWait,
		Table:   table,
		Where:   &where,
		Columns: columns,
		Until:   until,
		Rows:    rows,
		Timeout: timeout,
	}
}

// CommentOp builds a comment operation.  ovsdb-server logs the comment along
// with the transaction, which makes for a good audit trail
func CommentOp(comment string) Operation {
	return Operation{
		Op:      OpComment,
		Comment: comment,
	}
}

// AssertOp builds an assert operation
func AssertOp(lock string) Operation {
	return Operation{
		Op:   OpAssert,
		Lock: lock,
	}
}

// CommitOp builds a commit operation
func CommitOp(durable bool) Operation {
	return Operation{
		Op:      OpCommit,
		Durable: &durable,
	}
}

// AbortOp builds an abort operation
func AbortOp() Operation {
	return Operation{Op: OpAbort}
}

// CheckOperationResults reports the first error the remote found with the
// operations of a transaction.
//
// rfc7047 5.2 has it that the result array is at most as long as the
// operation array, that a short array means the remaining operations were
// never attempted, and that one extra element may be appended to describe a
// failure of the transaction as a whole
func CheckOperationResults(results []OperationResult, ops []Operation) error {
	for i := range results {
		result := &results[i]
		if result.Error == "" {
			continue
		}
		if i >= len(ops) {
			return errors.Wrapf(ErrOperation, "transaction: %s: %s", result.Error, result.Details)
		}
		op := &ops[i]
		return errors.Wrapf(ErrOperation, "op %d %s %s: %s: %s",
			i, op.Op, op.Table, result.Error, result.Details)
	}
	if len(results) < len(ops) {
		op := &ops[len(results)]
		return errors.Wrapf(ErrOperation, "only %d of %d operations were attempted, op %d is %s %s",
			len(results), len(ops), len(results), op.Op, op.Table)
	}
	return nil
}
