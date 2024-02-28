/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

const (
	LOB_FLAG_BYTE = 0
	LOB_FLAG_CHAR = 1

	LOB_IN_ROW  = 0x1
	LOB_OFF_ROW = 0x2

	NBLOB_HEAD_IN_ROW_FLAG = 0
	NBLOB_HEAD_BLOBID      = NBLOB_HEAD_IN_ROW_FLAG + BYTE_SIZE
	NBLOB_HEAD_BLOB_LEN    = NBLOB_HEAD_BLOBID + DDWORD_SIZE

	NBLOB_HEAD_OUTROW_GROUPID = NBLOB_HEAD_BLOB_LEN + ULINT_SIZE
	NBLOB_HEAD_OUTROW_FILEID  = NBLOB_HEAD_OUTROW_GROUPID + USINT_SIZE
	NBLOB_HEAD_OUTROW_PAGENO  = NBLOB_HEAD_OUTROW_FILEID + USINT_SIZE

	NBLOB_EX_HEAD_TABLE_ID   = NBLOB_HEAD_OUTROW_PAGENO + ULINT_SIZE
	NBLOB_EX_HEAD_COL_ID     = NBLOB_EX_HEAD_TABLE_ID + ULINT_SIZE
	NBLOB_EX_HEAD_ROW_ID     = NBLOB_EX_HEAD_COL_ID + USINT_SIZE
	NBLOB_EX_HEAD_FPA_GRPID  = NBLOB_EX_HEAD_ROW_ID + LINT64_SIZE
	NBLOB_EX_HEAD_FPA_FILEID = NBLOB_EX_HEAD_FPA_GRPID + USINT_SIZE
	NBLOB_EX_HEAD_FPA_PAGENO = NBLOB_EX_HEAD_FPA_FILEID + USINT_SIZE
	NBLOB_EX_HEAD_SIZE       = NBLOB_EX_HEAD_FPA_PAGENO + ULINT_SIZE

	NBLOB_OUTROW_HEAD_SIZE = NBLOB_HEAD_OUTROW_PAGENO + ULINT_SIZE

	NBLOB_INROW_HEAD_SIZE = NBLOB_HEAD_BLOB_LEN + ULINT_SIZE
)

type lob struct {
	blobId int64
	inRow  bool

	groupId   int16
	fileId    int16
	pageNo    int32
	tabId     int32
	colId     int16
	rowId     int64
	exGroupId int16
	exFileId  int16
	exPageNo  int32

	curFileId     int16
	curPageNo     int32
	curPageOffset int16
	totalOffset   int32
	readOver      bool

	connection       *DmConnection
	local            bool
	updateable       bool
	lobFlag          int8
	length           int64
	compatibleOracle bool
	fetchAll         bool
	freed            bool
	modify           bool

	Valid bool
}

func (lob *lob) GetLength() (int64, error) {
	var err error
	if err = lob.checkValid(); err != nil {
		return -1, err
	}
	if err = lob.checkFreed(); err != nil {
		return -1, err
	}
	if lob.length == -1 {

		if lob.length, err = lob.connection.Access.dm_build_1508(lob); err != nil {
			return -1, err
		}
	}
	return lob.length, nil
}

func (lob *lob) resetCurrentInfo() {
	lob.curFileId = lob.fileId
	lob.curPageNo = lob.pageNo
	lob.totalOffset = 0
	lob.curPageOffset = 0
}

func (lob *lob) getLengthFromHead(head []byte) int64 {
	return int64(Dm_build_650.Dm_build_752(head, NBLOB_HEAD_BLOB_LEN))
}

func (lob *lob) canOptimized(connection *DmConnection) bool {
	return !(lob.inRow || lob.fetchAll || lob.local || connection != lob.connection)
}

func (lob *lob) buildCtlData() (bytes []byte) {
	if lob.connection.NewLobFlag {
		bytes = make([]byte, NBLOB_EX_HEAD_SIZE, NBLOB_EX_HEAD_SIZE)
	} else {
		bytes = make([]byte, NBLOB_OUTROW_HEAD_SIZE, NBLOB_OUTROW_HEAD_SIZE)
	}
	Dm_build_650.Dm_build_651(bytes, NBLOB_HEAD_IN_ROW_FLAG, LOB_OFF_ROW)
	Dm_build_650.Dm_build_671(bytes, NBLOB_HEAD_BLOBID, lob.blobId)
	Dm_build_650.Dm_build_666(bytes, NBLOB_HEAD_BLOB_LEN, -1)

	Dm_build_650.Dm_build_661(bytes, NBLOB_HEAD_OUTROW_GROUPID, lob.groupId)
	Dm_build_650.Dm_build_661(bytes, NBLOB_HEAD_OUTROW_FILEID, lob.fileId)
	Dm_build_650.Dm_build_666(bytes, NBLOB_HEAD_OUTROW_PAGENO, lob.pageNo)

	if lob.connection.NewLobFlag {
		Dm_build_650.Dm_build_666(bytes, NBLOB_EX_HEAD_TABLE_ID, lob.tabId)
		Dm_build_650.Dm_build_661(bytes, NBLOB_EX_HEAD_COL_ID, lob.colId)
		Dm_build_650.Dm_build_671(bytes, NBLOB_EX_HEAD_ROW_ID, lob.rowId)
		Dm_build_650.Dm_build_661(bytes, NBLOB_EX_HEAD_FPA_GRPID, lob.exGroupId)
		Dm_build_650.Dm_build_661(bytes, NBLOB_EX_HEAD_FPA_FILEID, lob.exFileId)
		Dm_build_650.Dm_build_666(bytes, NBLOB_EX_HEAD_FPA_PAGENO, lob.exPageNo)
	}
	return
}

func (lob *lob) checkFreed() (err error) {
	if lob.freed {
		err = ECGO_LOB_FREED.throw()
	}
	return
}

func (lob *lob) checkValid() error {
	if !lob.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}
