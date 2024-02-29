/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"database/sql/driver"
	"io"
	"reflect"
	"strings"
)

type DmRows struct {
	filterable
	CurrentRows *innerRows
	finish      func()
}

func (r *DmRows) Columns() []string {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return nil
	}
	if len(r.filterChain.filters) == 0 {
		return r.columns()
	}
	return r.filterChain.reset().DmRowsColumns(r)
}

func (r *DmRows) Close() error {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return err
	}
	if len(r.filterChain.filters) == 0 {
		return r.close()
	}
	return r.filterChain.reset().DmRowsClose(r)
}

func (r *DmRows) Next(dest []driver.Value) error {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return err
	}
	if len(r.filterChain.filters) == 0 {
		return r.next(dest)
	}
	return r.filterChain.reset().DmRowsNext(r, dest)
}

func (r *DmRows) HasNextResultSet() bool {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return false
	}
	if len(r.filterChain.filters) == 0 {
		return r.hasNextResultSet()
	}
	return r.filterChain.reset().DmRowsHasNextResultSet(r)
}

func (r *DmRows) NextResultSet() error {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return err
	}
	if len(r.filterChain.filters) == 0 {
		return r.nextResultSet()
	}
	return r.filterChain.reset().DmRowsNextResultSet(r)
}

func (r *DmRows) ColumnTypeScanType(index int) reflect.Type {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return nil
	}
	if len(r.filterChain.filters) == 0 {
		return r.columnTypeScanType(index)
	}
	return r.filterChain.reset().DmRowsColumnTypeScanType(r, index)
}

func (r *DmRows) ColumnTypeDatabaseTypeName(index int) string {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return ""
	}
	if len(r.filterChain.filters) == 0 {
		return r.columnTypeDatabaseTypeName(index)
	}
	return r.filterChain.reset().DmRowsColumnTypeDatabaseTypeName(r, index)
}

func (r *DmRows) ColumnTypeLength(index int) (length int64, ok bool) {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return -1, false
	}
	if len(r.filterChain.filters) == 0 {
		return r.columnTypeLength(index)
	}
	return r.filterChain.reset().DmRowsColumnTypeLength(r, index)
}

func (r *DmRows) ColumnTypeNullable(index int) (nullable, ok bool) {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return false, false
	}
	if len(r.filterChain.filters) == 0 {
		return r.columnTypeNullable(index)
	}
	return r.filterChain.reset().DmRowsColumnTypeNullable(r, index)
}

func (r *DmRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	if err := r.CurrentRows.dmStmt.checkClosed(); err != nil {
		return -1, -1, false
	}
	if len(r.filterChain.filters) == 0 {
		return r.columnTypePrecisionScale(index)
	}
	return r.filterChain.reset().DmRowsColumnTypePrecisionScale(r, index)
}

func (dest *DmRows) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*dest = *new(DmRows)
		return nil
	case *DmRows:
		*dest = *src
		return nil
	default:
		return UNSUPPORTED_SCAN
	}
}

func (rows *DmRows) columns() []string {
	return rows.CurrentRows.Columns()
}

func (rows *DmRows) close() error {
	if f := rows.finish; f != nil {
		f()
		rows.finish = nil
	}
	return rows.CurrentRows.Close()
}

func (rows *DmRows) next(dest []driver.Value) error {
	return rows.CurrentRows.Next(dest)
}

func (rows *DmRows) hasNextResultSet() bool {
	return rows.CurrentRows.HasNextResultSet()
}

func (rows *DmRows) nextResultSet() error {
	return rows.CurrentRows.NextResultSet()
}

func (rows *DmRows) columnTypeScanType(index int) reflect.Type {
	return rows.CurrentRows.ColumnTypeScanType(index)
}

func (rows *DmRows) columnTypeDatabaseTypeName(index int) string {
	return rows.CurrentRows.ColumnTypeDatabaseTypeName(index)
}

func (rows *DmRows) columnTypeLength(index int) (length int64, ok bool) {
	return rows.CurrentRows.ColumnTypeLength(index)
}

func (rows *DmRows) columnTypeNullable(index int) (nullable, ok bool) {
	return rows.CurrentRows.ColumnTypeNullable(index)
}

func (rows *DmRows) columnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	return rows.CurrentRows.ColumnTypePrecisionScale(index)
}

type innerRows struct {
	dmStmt *DmStatement

	id int16

	columns []column

	datas [][][]byte

	datasOffset int

	datasStartPos int64

	currentPos int64

	totalRowCount int64

	fetchSize int

	sizeOfRow int

	isBdta bool

	nextExecInfo *execRetInfo

	next *innerRows

	dmRows *DmRows

	closed bool
}

func (innerRows *innerRows) checkClosed() error {
	if innerRows.closed {
		return ECGO_RESULTSET_CLOSED.throw()
	}
	return nil
}

func (innerRows *innerRows) Columns() []string {
	if err := innerRows.checkClosed(); err != nil {
		return nil
	}

	columnNames := make([]string, len(innerRows.columns))
	nameCase := innerRows.dmStmt.dmConn.dmConnector.columnNameCase

	for i, column := range innerRows.columns {
		if nameCase == COLUMN_NAME_NATURAL_CASE {
			columnNames[i] = column.name
		} else if nameCase == COLUMN_NAME_UPPER_CASE {
			columnNames[i] = strings.ToUpper(column.name)
		} else if nameCase == COLUMN_NAME_LOWER_CASE {
			columnNames[i] = strings.ToLower(column.name)
		} else {
			columnNames[i] = column.name
		}
	}

	return columnNames
}

func (innerRows *innerRows) Close() error {
	if innerRows.closed {
		return nil
	}

	innerRows.closed = true

	if innerRows.dmStmt.innerUsed {
		innerRows.dmStmt.close()
	} else {
		delete(innerRows.dmStmt.rsMap, innerRows.id)
	}

	innerRows.dmStmt = nil

	return nil
}

func (innerRows *innerRows) Next(dest []driver.Value) error {
	err := innerRows.checkClosed()
	if err != nil {
		return err
	}

	if innerRows.totalRowCount == 0 || innerRows.currentPos >= innerRows.totalRowCount {
		return io.EOF
	}

	if innerRows.currentPos+1 == innerRows.totalRowCount {
		innerRows.currentPos++
		innerRows.datasOffset++
		return io.EOF
	}

	if innerRows.currentPos+1 < innerRows.datasStartPos || innerRows.currentPos+1 >= innerRows.datasStartPos+int64(len(innerRows.datas)) {
		if innerRows.fetchData(innerRows.currentPos + 1) {
			innerRows.currentPos++
			err := innerRows.getRowData(dest)
			if err != nil {
				return err
			}
		} else {
			innerRows.currentPos++
			innerRows.datasOffset++
			return io.EOF
		}
	} else {
		innerRows.currentPos++
		innerRows.datasOffset++
		err := innerRows.getRowData(dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (innerRows *innerRows) HasNextResultSet() bool {
	err := innerRows.checkClosed()
	if err != nil {
		return false
	}

	if innerRows.nextExecInfo != nil {
		return innerRows.nextExecInfo.hasResultSet
	}

	innerRows.nextExecInfo, err = innerRows.dmStmt.dmConn.Access.Dm_build_1466(innerRows.dmStmt, 0)
	if err != nil {
		return false
	}

	if innerRows.nextExecInfo.hasResultSet {
		innerRows.next = newInnerRows(innerRows.id+1, innerRows.dmStmt, innerRows.nextExecInfo)
		return true
	}

	return false
}

func (innerRows *innerRows) NextResultSet() error {
	err := innerRows.checkClosed()
	if err != nil {
		return err
	}

	if innerRows.nextExecInfo == nil {
		innerRows.HasNextResultSet()
	}

	if innerRows.next == nil {
		return io.EOF
	}

	innerRows.next.dmRows = innerRows.dmRows
	innerRows.dmRows.CurrentRows = innerRows.next
	return nil
}

func (innerRows *innerRows) ColumnTypeScanType(index int) reflect.Type {
	if err := innerRows.checkClosed(); err != nil {
		return nil
	}
	if column := innerRows.checkIndex(index); column != nil {
		return column.ScanType()
	}
	return nil
}

func (innerRows *innerRows) ColumnTypeDatabaseTypeName(index int) string {
	if err := innerRows.checkClosed(); err != nil {
		return ""
	}
	if column := innerRows.checkIndex(index); column != nil {
		return column.typeName
	}
	return ""
}

func (innerRows *innerRows) ColumnTypeLength(index int) (length int64, ok bool) {
	if err := innerRows.checkClosed(); err != nil {
		return 0, false
	}
	if column := innerRows.checkIndex(index); column != nil {
		return column.Length()
	}
	return 0, false
}

func (innerRows *innerRows) ColumnTypeNullable(index int) (nullable, ok bool) {
	if err := innerRows.checkClosed(); err != nil {
		return false, false
	}
	if column := innerRows.checkIndex(index); column != nil {
		return column.nullable, true
	}
	return false, false
}

func (innerRows *innerRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	if err := innerRows.checkClosed(); err != nil {
		return 0, 0, false
	}
	if column := innerRows.checkIndex(index); column != nil {
		return column.PrecisionScale()
	}
	return 0, 0, false
}

func newDmRows(currentRows *innerRows) *DmRows {
	dr := new(DmRows)
	dr.resetFilterable(&currentRows.dmStmt.filterable)
	dr.CurrentRows = currentRows
	dr.idGenerator = dmRowsIDGenerator
	currentRows.dmRows = dr
	return dr
}

func newInnerRows(id int16, stmt *DmStatement, execInfo *execRetInfo) *innerRows {
	rows := new(innerRows)
	rows.id = id
	rows.dmStmt = stmt
	rows.columns = stmt.columns
	rows.datas = execInfo.rsDatas
	rows.totalRowCount = execInfo.updateCount
	rows.isBdta = execInfo.rsBdta
	rows.fetchSize = stmt.fetchSize

	if len(execInfo.rsDatas) == 0 {
		rows.sizeOfRow = 0
	} else {
		rows.sizeOfRow = execInfo.rsSizeof / len(execInfo.rsDatas)
	}

	rows.currentPos = -1
	rows.datasOffset = -1
	rows.datasStartPos = 0

	rows.nextExecInfo = nil
	rows.next = nil

	if rows.dmStmt.rsMap != nil {
		rows.dmStmt.rsMap[rows.id] = rows
	}

	if stmt.dmConn.dmConnector.enRsCache && execInfo.rsCacheOffset > 0 &&
		int64(len(execInfo.rsDatas)) == execInfo.updateCount {
		rp.put(stmt, stmt.nativeSql, execInfo)
	}

	return rows
}

func newLocalInnerRows(stmt *DmStatement, columns []column, rsDatas [][][]byte) *innerRows {
	rows := new(innerRows)
	rows.id = 0
	rows.dmStmt = stmt
	rows.fetchSize = stmt.fetchSize

	if columns == nil {
		rows.columns = make([]column, 0)
	} else {
		rows.columns = columns
	}

	if rsDatas == nil {
		rows.datas = make([][][]byte, 0)
		rows.totalRowCount = 0
	} else {
		rows.datas = rsDatas
		rows.totalRowCount = int64(len(rsDatas))
	}

	rows.isBdta = false
	return rows
}

func (innerRows *innerRows) checkIndex(index int) *column {
	if index < 0 || index > len(innerRows.columns)-1 {
		return nil
	}

	return &innerRows.columns[index]
}

func (innerRows *innerRows) fetchData(startPos int64) bool {
	execInfo, err := innerRows.dmStmt.dmConn.Access.Dm_build_1473(innerRows, startPos)
	if err != nil {
		return false
	}

	innerRows.totalRowCount = execInfo.updateCount
	if execInfo.rsDatas != nil {
		innerRows.datas = execInfo.rsDatas
		innerRows.datasStartPos = startPos
		innerRows.datasOffset = 0
		return true
	}

	return false
}

func (innerRows *innerRows) getRowData(dest []driver.Value) (err error) {
	for i, column := range innerRows.columns {

		if i <= len(dest)-1 {
			if column.colType == CURSOR {
				var tmpExecInfo *execRetInfo
				tmpExecInfo, err = innerRows.dmStmt.dmConn.Access.Dm_build_1466(innerRows.dmStmt, 1)
				if err != nil {
					return err
				}

				if tmpExecInfo.hasResultSet {
					dest[i] = newDmRows(newInnerRows(innerRows.id+1, innerRows.dmStmt, tmpExecInfo))
				} else {
					dest[i] = nil
				}
				continue
			}

			dest[i], err = column.getColumnData(innerRows.datas[innerRows.datasOffset][i+1], innerRows.dmStmt.dmConn)
			innerRows.columns[i].isBdta = innerRows.isBdta
			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	return nil
}

func (innerRows *innerRows) getRowCount() int64 {
	innerRows.checkClosed()

	if innerRows.totalRowCount == INT64_MAX {
		return -1
	}

	return innerRows.totalRowCount
}
