/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"database/sql/driver"
	"io"
)

type DmClob struct {
	lob
	data           []rune
	serverEncoding string
}

func newDmClob() *DmClob {
	return &DmClob{
		lob: lob{
			inRow:            true,
			groupId:          -1,
			fileId:           -1,
			pageNo:           -1,
			readOver:         false,
			local:            true,
			updateable:       true,
			length:           -1,
			compatibleOracle: false,
			fetchAll:         false,
			freed:            false,
			modify:           false,
			Valid:            true,
		},
	}
}

func newClobFromDB(value []byte, conn *DmConnection, column *column, fetchAll bool) *DmClob {
	var clob = newDmClob()
	clob.connection = conn
	clob.lobFlag = LOB_FLAG_CHAR
	clob.compatibleOracle = conn.CompatibleOracle()
	clob.local = false
	clob.updateable = !column.readonly
	clob.tabId = column.lobTabId
	clob.colId = column.lobColId

	clob.inRow = Dm_build_650.Dm_build_743(value, NBLOB_HEAD_IN_ROW_FLAG) == LOB_IN_ROW
	clob.blobId = Dm_build_650.Dm_build_757(value, NBLOB_HEAD_BLOBID)
	if !clob.inRow {
		clob.groupId = Dm_build_650.Dm_build_747(value, NBLOB_HEAD_OUTROW_GROUPID)
		clob.fileId = Dm_build_650.Dm_build_747(value, NBLOB_HEAD_OUTROW_FILEID)
		clob.pageNo = Dm_build_650.Dm_build_752(value, NBLOB_HEAD_OUTROW_PAGENO)
	}
	if conn.NewLobFlag {
		clob.tabId = Dm_build_650.Dm_build_752(value, NBLOB_EX_HEAD_TABLE_ID)
		clob.colId = Dm_build_650.Dm_build_747(value, NBLOB_EX_HEAD_COL_ID)
		clob.rowId = Dm_build_650.Dm_build_757(value, NBLOB_EX_HEAD_ROW_ID)
		clob.exGroupId = Dm_build_650.Dm_build_747(value, NBLOB_EX_HEAD_FPA_GRPID)
		clob.exFileId = Dm_build_650.Dm_build_747(value, NBLOB_EX_HEAD_FPA_FILEID)
		clob.exPageNo = Dm_build_650.Dm_build_752(value, NBLOB_EX_HEAD_FPA_PAGENO)
	}
	clob.resetCurrentInfo()

	clob.serverEncoding = conn.getServerEncoding()
	if clob.inRow {
		if conn.NewLobFlag {
			clob.data = []rune(Dm_build_650.Dm_build_807(value, NBLOB_EX_HEAD_SIZE, int(clob.getLengthFromHead(value)), clob.serverEncoding, conn))
		} else {
			clob.data = []rune(Dm_build_650.Dm_build_807(value, NBLOB_INROW_HEAD_SIZE, int(clob.getLengthFromHead(value)), clob.serverEncoding, conn))
		}
		clob.length = int64(len(clob.data))
	} else if fetchAll {
		clob.loadAllData()
	}
	return clob
}

func newClobOfLocal(value string, conn *DmConnection) *DmClob {
	var clob = newDmClob()
	clob.connection = conn
	clob.lobFlag = LOB_FLAG_CHAR
	clob.data = []rune(value)
	clob.length = int64(len(clob.data))
	return clob
}

func NewClob(value string) *DmClob {
	var clob = newDmClob()

	clob.lobFlag = LOB_FLAG_CHAR
	clob.data = []rune(value)
	clob.length = int64(len(clob.data))
	return clob
}

func (clob *DmClob) ReadString(pos int, length int) (result string, err error) {
	if err = clob.checkValid(); err != nil {
		return
	}
	result, err = clob.getSubString(int64(pos), int32(length))
	if err != nil {
		return
	}
	if len(result) == 0 {
		err = io.EOF
		return
	}
	return
}

func (clob *DmClob) WriteString(pos int, s string) (n int, err error) {
	if err = clob.checkValid(); err != nil {
		return
	}
	if err = clob.checkFreed(); err != nil {
		return
	}
	if pos < 1 {
		err = ECGO_INVALID_LENGTH_OR_OFFSET.throw()
		return
	}
	if !clob.updateable {
		err = ECGO_RESULTSET_IS_READ_ONLY.throw()
		return
	}
	pos -= 1
	if clob.local || clob.fetchAll {
		if int64(pos) > clob.length {
			err = ECGO_INVALID_LENGTH_OR_OFFSET.throw()
			return
		}
		clob.setLocalData(pos, s)
		n = len(s)
	} else {
		if err = clob.connection.checkClosed(); err != nil {
			return -1, err
		}
		var writeLen, err = clob.connection.Access.dm_build_1545(clob, pos, s, clob.serverEncoding)
		if err != nil {
			return -1, err
		}

		if clob.groupId == -1 {
			clob.setLocalData(pos, s)
		} else {
			clob.inRow = false
			clob.length = -1
		}
		n = writeLen
	}
	clob.modify = true
	return
}

func (clob *DmClob) Truncate(length int64) error {
	var err error
	if err = clob.checkValid(); err != nil {
		return err
	}
	if err = clob.checkFreed(); err != nil {
		return err
	}
	if length < 0 {
		return ECGO_INVALID_LENGTH_OR_OFFSET.throw()
	}
	if !clob.updateable {
		return ECGO_RESULTSET_IS_READ_ONLY.throw()
	}
	if clob.local || clob.fetchAll {
		if length >= int64(len(clob.data)) {
			return nil
		}
		clob.data = clob.data[0:length]
		clob.length = int64(len(clob.data))
	} else {
		if err = clob.connection.checkClosed(); err != nil {
			return err
		}
		clob.length, err = clob.connection.Access.dm_build_1575(&clob.lob, int(length))
		if err != nil {
			return err
		}
		if clob.groupId == -1 {
			clob.data = clob.data[0:clob.length]
		}
	}
	clob.modify = true
	return nil
}

func (dest *DmClob) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmClob)

		(*dest).Valid = false
		return nil
	case string:
		*dest = *NewClob(src)
		return nil
	case *DmClob:
		*dest = *src
		return nil
	default:
		return UNSUPPORTED_SCAN.throw()
	}
}

func (clob DmClob) Value() (driver.Value, error) {
	if !clob.Valid {
		return nil, nil
	}
	return clob, nil
}

func (clob *DmClob) getSubString(pos int64, len int32) (string, error) {
	var err error
	var leaveLength int64
	if err = clob.checkFreed(); err != nil {
		return "", err
	}
	if pos < 1 || len < 0 {
		return "", ECGO_INVALID_LENGTH_OR_OFFSET.throw()
	}
	pos = pos - 1
	if leaveLength, err = clob.GetLength(); err != nil {
		return "", err
	}
	if pos > leaveLength {
		pos = leaveLength
	}
	leaveLength -= pos
	if leaveLength < 0 {
		return "", ECGO_INVALID_LENGTH_OR_OFFSET.throw()
	}
	if int64(len) > leaveLength {
		len = int32(leaveLength)
	}
	if clob.local || clob.inRow || clob.fetchAll {
		if pos > clob.length {
			return "", ECGO_INVALID_LENGTH_OR_OFFSET.throw()
		}
		return string(clob.data[pos : pos+int64(len)]), nil
	} else {

		return clob.connection.Access.dm_build_1533(clob, int32(pos), len)
	}
}

func (clob *DmClob) loadAllData() {
	clob.checkFreed()
	if clob.local || clob.inRow || clob.fetchAll {
		return
	}
	len, _ := clob.GetLength()
	s, _ := clob.getSubString(1, int32(len))
	clob.data = []rune(s)
	clob.fetchAll = true
}

func (clob *DmClob) setLocalData(pos int, str string) {
	if pos+len(str) >= int(clob.length) {
		clob.data = []rune(string(clob.data[0:pos]) + str)
	} else {
		clob.data = []rune(string(clob.data[0:pos]) + str + string(clob.data[pos+len(str):len(clob.data)]))
	}
	clob.length = int64(len(clob.data))
}

func (d *DmClob) GormDataType() string {
	return "CLOB"
}
