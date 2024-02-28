/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"database/sql/driver"
)

const (
	OBJ_BLOB_MAGIC = 78111999

	CLTN_TYPE_IND_TABLE = 3

	CLTN_TYPE_NST_TABLE = 2

	CLTN_TYPE_VARRAY = 1
)

type TypeDescriptor struct {
	column *column

	m_sqlName *sqlName

	m_objId int

	m_objVersion int

	m_outerId int

	m_outerVer int

	m_subId int

	m_cltnType int

	m_maxCnt int

	m_length int

	m_size int

	m_conn *DmConnection

	m_serverEncoding string

	m_arrObj *TypeDescriptor

	m_fieldsObj []TypeDescriptor

	m_descBuf []byte
}

func newTypeDescriptorWithFulName(fulName string, conn *DmConnection) *TypeDescriptor {
	td := new(TypeDescriptor)
	td.init()
	td.m_sqlName = newSqlNameByFulName(fulName)
	td.m_conn = conn
	return td
}

func newTypeDescriptor(conn *DmConnection) *TypeDescriptor {
	td := new(TypeDescriptor)
	td.init()
	td.m_sqlName = newSqlNameByConn(conn)
	td.m_conn = conn
	return td
}

func (typeDescriptor *TypeDescriptor) init() {
	typeDescriptor.column = new(column).InitColumn()

	typeDescriptor.m_sqlName = nil

	typeDescriptor.m_objId = -1

	typeDescriptor.m_objVersion = -1

	typeDescriptor.m_outerId = 0

	typeDescriptor.m_outerVer = 0

	typeDescriptor.m_subId = 0

	typeDescriptor.m_cltnType = 0

	typeDescriptor.m_maxCnt = 0

	typeDescriptor.m_length = 0

	typeDescriptor.m_size = 0

	typeDescriptor.m_conn = nil

	typeDescriptor.m_serverEncoding = ""

	typeDescriptor.m_arrObj = nil

	typeDescriptor.m_fieldsObj = nil

	typeDescriptor.m_descBuf = nil
}

func (typeDescriptor *TypeDescriptor) parseDescByName() error {
	sql := "BEGIN ? = SF_DESCRIBE_TYPE(?); END;"

	params := make([]driver.Value, 2)
	params[1] = typeDescriptor.m_sqlName.m_fulName

	rs, err := typeDescriptor.m_conn.query(sql, params)
	if err != nil {
		return err
	}
	rs.close()
	l, err := params[0].(*DmBlob).GetLength()
	if err != nil {
		return err
	}

	buf, err := params[0].(*DmBlob).getBytes(1, int32(l))
	if err != nil {
		return err
	}
	typeDescriptor.m_serverEncoding = typeDescriptor.m_conn.getServerEncoding()
	err = typeDescriptor.unpack(Dm_build_1014(buf))
	if err != nil {
		return err
	}
	return nil
}

func (typeDescriptor *TypeDescriptor) getFulName() (string, error) {
	return typeDescriptor.m_sqlName.getFulName()
}

func (typeDescriptor *TypeDescriptor) getDType() int {
	return int(typeDescriptor.column.colType)
}

func (typeDescriptor *TypeDescriptor) getPrec() int {
	return int(typeDescriptor.column.prec)
}

func (typeDescriptor *TypeDescriptor) getScale() int {
	return int(typeDescriptor.column.scale)
}

func (typeDescriptor *TypeDescriptor) getServerEncoding() string {
	if typeDescriptor.m_serverEncoding == "" {
		return typeDescriptor.m_conn.getServerEncoding()
	} else {
		return typeDescriptor.m_serverEncoding
	}
}

func (typeDescriptor *TypeDescriptor) getObjId() int {
	return typeDescriptor.m_objId
}

func (typeDescriptor *TypeDescriptor) getStaticArrayLength() int {
	return typeDescriptor.m_length
}

func (typeDescriptor *TypeDescriptor) getStrctMemSize() int {
	return typeDescriptor.m_size
}

func (typeDescriptor *TypeDescriptor) getOuterId() int {
	return typeDescriptor.m_outerId
}

func (typeDescriptor *TypeDescriptor) getCltnType() int {
	return typeDescriptor.m_cltnType
}

func (typeDescriptor *TypeDescriptor) getMaxCnt() int {
	return typeDescriptor.m_maxCnt
}

func getPackSize(typeDesc *TypeDescriptor) (int, error) {
	len := 0

	switch typeDesc.column.colType {
	case ARRAY, SARRAY:
		return getPackArraySize(typeDesc)

	case CLASS:
		return getPackClassSize(typeDesc)

	case PLTYPE_RECORD:
		return getPackRecordSize(typeDesc)
	}

	len += ULINT_SIZE

	len += ULINT_SIZE

	len += ULINT_SIZE

	return len, nil
}

func pack(typeDesc *TypeDescriptor, msg *Dm_build_1009) error {
	switch typeDesc.column.colType {
	case ARRAY, SARRAY:
		return packArray(typeDesc, msg)
	case CLASS:
		return packClass(typeDesc, msg)
	case PLTYPE_RECORD:
		return packRecord(typeDesc, msg)
	}

	msg.Dm_build_1064(typeDesc.column.colType)

	msg.Dm_build_1064(typeDesc.column.prec)

	msg.Dm_build_1064(typeDesc.column.scale)
	return nil
}

func getPackArraySize(arrDesc *TypeDescriptor) (int, error) {
	l := 0

	l += ULINT_SIZE

	name := arrDesc.m_sqlName.m_name
	l += USINT_SIZE

	serverEncoding := arrDesc.getServerEncoding()
	ret := Dm_build_650.Dm_build_866(name, serverEncoding, arrDesc.m_conn)
	l += len(ret)

	l += ULINT_SIZE

	l += ULINT_SIZE

	l += ULINT_SIZE

	i, err := getPackSize(arrDesc.m_arrObj)
	if err != nil {
		return 0, err
	}

	l += i

	return l, nil
}

func packArray(arrDesc *TypeDescriptor, msg *Dm_build_1009) error {

	msg.Dm_build_1064(arrDesc.column.colType)

	msg.Dm_build_1120(arrDesc.m_sqlName.m_name, arrDesc.getServerEncoding(), arrDesc.m_conn)

	msg.Dm_build_1064(int32(arrDesc.m_objId))

	msg.Dm_build_1064(int32(arrDesc.m_objVersion))

	msg.Dm_build_1064(int32(arrDesc.m_length))

	return pack(arrDesc.m_arrObj, msg)
}

func packRecord(strctDesc *TypeDescriptor, msg *Dm_build_1009) error {

	msg.Dm_build_1064(strctDesc.column.colType)

	msg.Dm_build_1120(strctDesc.m_sqlName.m_name, strctDesc.getServerEncoding(), strctDesc.m_conn)

	msg.Dm_build_1064(int32(strctDesc.m_objId))

	msg.Dm_build_1064(int32(strctDesc.m_objVersion))

	msg.Dm_build_1060(int16(strctDesc.m_size))

	for i := 0; i < strctDesc.m_size; i++ {
		err := pack(&strctDesc.m_fieldsObj[i], msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func getPackRecordSize(strctDesc *TypeDescriptor) (int, error) {
	l := 0

	l += ULINT_SIZE

	name := strctDesc.m_sqlName.m_name
	l += USINT_SIZE

	serverEncoding := strctDesc.getServerEncoding()
	ret := Dm_build_650.Dm_build_866(name, serverEncoding, strctDesc.m_conn)
	l += len(ret)

	l += ULINT_SIZE

	l += ULINT_SIZE

	l += USINT_SIZE

	for i := 0; i < strctDesc.m_size; i++ {
		i, err := getPackSize(&strctDesc.m_fieldsObj[i])
		if err != nil {
			return 0, err
		}
		l += i
	}

	return l, nil
}

func getPackClassSize(strctDesc *TypeDescriptor) (int, error) {
	l := 0

	l += ULINT_SIZE

	name := strctDesc.m_sqlName.m_name
	l += USINT_SIZE

	serverEncoding := strctDesc.getServerEncoding()
	ret := Dm_build_650.Dm_build_866(name, serverEncoding, strctDesc.m_conn)
	l += len(ret)

	l += ULINT_SIZE

	l += ULINT_SIZE

	if strctDesc.m_objId == 4 {

		l += ULINT_SIZE

		l += ULINT_SIZE

		l += USINT_SIZE
	}

	return l, nil
}

func packClass(strctDesc *TypeDescriptor, msg *Dm_build_1009) error {

	msg.Dm_build_1064(strctDesc.column.colType)

	msg.Dm_build_1120(strctDesc.m_sqlName.m_name, strctDesc.getServerEncoding(), strctDesc.m_conn)

	msg.Dm_build_1064(int32(strctDesc.m_objId))

	msg.Dm_build_1064(int32(strctDesc.m_objVersion))

	if strctDesc.m_objId == 4 {

		msg.Dm_build_1064(int32(strctDesc.m_outerId))

		msg.Dm_build_1064(int32(strctDesc.m_outerVer))

		msg.Dm_build_1064(int32(strctDesc.m_subId))

	}

	return nil
}

func (typeDescriptor *TypeDescriptor) unpack(buffer *Dm_build_1009) error {

	typeDescriptor.column.colType = buffer.Dm_build_1138()

	switch typeDescriptor.column.colType {
	case ARRAY, SARRAY:
		return typeDescriptor.unpackArray(buffer)
	case CLASS:
		return typeDescriptor.unpackClass(buffer)
	case PLTYPE_RECORD:
		return typeDescriptor.unpackRecord(buffer)
	}

	typeDescriptor.column.prec = buffer.Dm_build_1138()

	typeDescriptor.column.scale = buffer.Dm_build_1138()
	return nil
}

func (typeDescriptor *TypeDescriptor) unpackArray(buffer *Dm_build_1009) error {

	typeDescriptor.m_sqlName.m_name = buffer.Dm_build_1188(typeDescriptor.getServerEncoding(), typeDescriptor.m_conn)

	typeDescriptor.m_sqlName.m_schId = int(buffer.Dm_build_1138())

	typeDescriptor.m_sqlName.m_packId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objVersion = int(buffer.Dm_build_1138())

	typeDescriptor.m_length = int(buffer.Dm_build_1138())
	if typeDescriptor.column.colType == ARRAY {
		typeDescriptor.m_length = 0
	}

	typeDescriptor.m_arrObj = newTypeDescriptor(typeDescriptor.m_conn)
	return typeDescriptor.m_arrObj.unpack(buffer)
}

func (typeDescriptor *TypeDescriptor) unpackRecord(buffer *Dm_build_1009) error {

	typeDescriptor.m_sqlName.m_name = buffer.Dm_build_1188(typeDescriptor.getServerEncoding(), typeDescriptor.m_conn)

	typeDescriptor.m_sqlName.m_schId = int(buffer.Dm_build_1138())

	typeDescriptor.m_sqlName.m_packId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objVersion = int(buffer.Dm_build_1138())

	typeDescriptor.m_size = int(buffer.Dm_build_1153())

	typeDescriptor.m_fieldsObj = make([]TypeDescriptor, typeDescriptor.m_size)
	for i := 0; i < typeDescriptor.m_size; i++ {
		typeDescriptor.m_fieldsObj[i] = *newTypeDescriptor(typeDescriptor.m_conn)
		typeDescriptor.m_fieldsObj[i].unpack(buffer)
	}

	return nil
}

func (typeDescriptor *TypeDescriptor) unpackClnt_nestTab(buffer *Dm_build_1009) error {

	typeDescriptor.m_maxCnt = int(buffer.Dm_build_1138())

	typeDescriptor.m_arrObj = newTypeDescriptor(typeDescriptor.m_conn)

	typeDescriptor.m_arrObj.unpack(buffer)

	return nil
}

func (typeDescriptor *TypeDescriptor) unpackClnt(buffer *Dm_build_1009) error {

	typeDescriptor.m_outerId = int(buffer.Dm_build_1138())

	typeDescriptor.m_outerVer = int(buffer.Dm_build_1138())

	typeDescriptor.m_subId = int(buffer.Dm_build_1153())

	typeDescriptor.m_cltnType = int(buffer.Dm_build_1153())

	switch typeDescriptor.m_cltnType {
	case CLTN_TYPE_IND_TABLE:
		return ECGO_UNSUPPORTED_TYPE.throw()

	case CLTN_TYPE_NST_TABLE, CLTN_TYPE_VARRAY:
		return typeDescriptor.unpackClnt_nestTab(buffer)

	}
	return nil
}

func (typeDescriptor *TypeDescriptor) unpackClass(buffer *Dm_build_1009) error {

	typeDescriptor.m_sqlName.m_name = buffer.Dm_build_1188(typeDescriptor.getServerEncoding(), typeDescriptor.m_conn)

	typeDescriptor.m_sqlName.m_schId = int(buffer.Dm_build_1138())

	typeDescriptor.m_sqlName.m_packId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objId = int(buffer.Dm_build_1138())

	typeDescriptor.m_objVersion = int(buffer.Dm_build_1138())

	if typeDescriptor.m_objId == 4 {
		return typeDescriptor.unpackClnt(buffer)
	} else {

		typeDescriptor.m_size = int(buffer.Dm_build_1153())

		typeDescriptor.m_fieldsObj = make([]TypeDescriptor, typeDescriptor.m_size)
		for i := 0; i < typeDescriptor.m_size; i++ {
			typeDescriptor.m_fieldsObj[i] = *newTypeDescriptor(typeDescriptor.m_conn)
			err := typeDescriptor.m_fieldsObj[i].unpack(buffer)
			if err != nil {
				return err
			}
		}
		return nil
	}

}

func calcChkDescLen_array(desc *TypeDescriptor) (int, error) {
	offset := 0

	offset += USINT_SIZE

	offset += ULINT_SIZE

	tmp, err := calcChkDescLen(desc)
	if err != nil {
		return 0, err
	}

	offset += tmp

	return offset, nil
}

func calcChkDescLen_record(desc *TypeDescriptor) (int, error) {
	offset := 0

	offset += USINT_SIZE

	offset += USINT_SIZE

	for i := 0; i < desc.m_size; i++ {
		tmp, err := calcChkDescLen(&desc.m_fieldsObj[i])
		if err != nil {
			return 0, err
		}
		offset += tmp
	}

	return offset, nil
}

func calcChkDescLen_class_normal(desc *TypeDescriptor) (int, error) {
	offset := 0

	offset += USINT_SIZE

	for i := 0; i < desc.m_size; i++ {
		tmp, err := calcChkDescLen(&desc.m_fieldsObj[i])
		if err != nil {
			return 0, err
		}
		offset += tmp
	}

	return offset, nil
}

func calcChkDescLen_class_cnlt(desc *TypeDescriptor) (int, error) {
	offset := 0

	offset += USINT_SIZE

	offset += ULINT_SIZE

	switch desc.getCltnType() {
	case CLTN_TYPE_IND_TABLE:
		return 0, ECGO_UNSUPPORTED_TYPE.throw()

	case CLTN_TYPE_VARRAY, CLTN_TYPE_NST_TABLE:

		i, err := calcChkDescLen(desc.m_arrObj)
		if err != nil {
			return 0, err
		}

		offset += i
	}

	return offset, nil
}

func calcChkDescLen_class(desc *TypeDescriptor) (int, error) {
	offset := 0

	offset += USINT_SIZE

	offset += BYTE_SIZE

	if desc.m_objId == 4 {
		i, err := calcChkDescLen_class_cnlt(desc)
		if err != nil {
			return 0, err
		}
		offset += i
	} else {
		i, err := calcChkDescLen_class_normal(desc)
		if err != nil {
			return 0, err
		}
		offset += i
	}

	return offset, nil
}

func calcChkDescLen_buildin() int {
	offset := 0

	offset += USINT_SIZE

	offset += USINT_SIZE

	offset += USINT_SIZE

	return offset
}

func calcChkDescLen(desc *TypeDescriptor) (int, error) {

	switch desc.getDType() {
	case ARRAY, SARRAY:
		return calcChkDescLen_array(desc)

	case PLTYPE_RECORD:
		return calcChkDescLen_record(desc)

	case CLASS:
		return calcChkDescLen_class(desc)

	default:
		return calcChkDescLen_buildin(), nil
	}

}

func (typeDescriptor *TypeDescriptor) makeChkDesc_array(offset int, desc *TypeDescriptor) (int, error) {

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, ARRAY)
	offset += USINT_SIZE

	Dm_build_650.Dm_build_666(typeDescriptor.m_descBuf, offset, int32(desc.m_length))
	offset += ULINT_SIZE

	return typeDescriptor.makeChkDesc(offset, desc)
}

func (typeDescriptor *TypeDescriptor) makeChkDesc_record(offset int, desc *TypeDescriptor) (int, error) {

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, PLTYPE_RECORD)
	offset += USINT_SIZE

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, int16(desc.m_size))
	offset += USINT_SIZE
	var err error
	for i := 0; i < desc.m_size; i++ {
		offset, err = typeDescriptor.makeChkDesc(offset, &desc.m_fieldsObj[i])
		if err != nil {
			return 0, err
		}
	}

	return offset, nil
}

func (typeDescriptor *TypeDescriptor) makeChkDesc_buildin(offset int, desc *TypeDescriptor) int {
	dtype := int16(desc.getDType())
	prec := 0
	scale := 0

	if dtype != BLOB {
		prec = desc.getPrec()
		scale = desc.getScale()
	}

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, dtype)
	offset += USINT_SIZE

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, int16(prec))
	offset += USINT_SIZE

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, int16(scale))
	offset += USINT_SIZE

	return offset
}

func (typeDescriptor *TypeDescriptor) makeChkDesc_class_normal(offset int, desc *TypeDescriptor) (int, error) {

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, int16(desc.m_size))
	offset += USINT_SIZE
	var err error

	for i := 0; i < desc.m_size; i++ {
		offset, err = typeDescriptor.makeChkDesc(offset, &desc.m_fieldsObj[i])
		if err != nil {
			return 0, err
		}
	}

	return offset, nil
}

func (typeDescriptor *TypeDescriptor) makeChkDesc_class_clnt(offset int, desc *TypeDescriptor) (int, error) {

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, int16(desc.m_cltnType))
	offset += USINT_SIZE

	Dm_build_650.Dm_build_666(typeDescriptor.m_descBuf, offset, int32(desc.getMaxCnt()))
	offset += ULINT_SIZE

	switch desc.m_cltnType {
	case CLTN_TYPE_IND_TABLE:
		return 0, ECGO_UNSUPPORTED_TYPE.throw()

	case CLTN_TYPE_NST_TABLE, CLTN_TYPE_VARRAY:

		return typeDescriptor.makeChkDesc(offset, desc.m_arrObj)
	}

	return offset, nil
}

func (typeDescriptor *TypeDescriptor) makeChkDesc_class(offset int, desc *TypeDescriptor) (int, error) {

	Dm_build_650.Dm_build_661(typeDescriptor.m_descBuf, offset, CLASS)
	offset += USINT_SIZE

	isClnt := false
	if desc.m_objId == 4 {
		isClnt = true
	}

	if isClnt {
		Dm_build_650.Dm_build_651(typeDescriptor.m_descBuf, offset, byte(1))
	} else {
		Dm_build_650.Dm_build_651(typeDescriptor.m_descBuf, offset, byte(0))
	}

	offset += BYTE_SIZE

	if isClnt {
		return typeDescriptor.makeChkDesc_class_clnt(offset, desc)
	} else {
		return typeDescriptor.makeChkDesc_class_normal(offset, desc)
	}
}

func (typeDescriptor *TypeDescriptor) makeChkDesc(offset int, subDesc *TypeDescriptor) (int, error) {
	switch subDesc.getDType() {
	case ARRAY, SARRAY:
		return typeDescriptor.makeChkDesc_array(offset, subDesc)

	case PLTYPE_RECORD:
		return typeDescriptor.makeChkDesc_record(offset, subDesc)

	case CLASS:
		return typeDescriptor.makeChkDesc_class(offset, subDesc)

	default:
		return typeDescriptor.makeChkDesc_buildin(offset, subDesc), nil
	}

}

func (typeDescriptor *TypeDescriptor) getClassDescChkInfo() ([]byte, error) {
	if typeDescriptor.m_descBuf != nil {
		return typeDescriptor.m_descBuf, nil
	}

	l, err := calcChkDescLen(typeDescriptor)
	if err != nil {
		return nil, err
	}
	typeDescriptor.m_descBuf = make([]byte, l)

	typeDescriptor.makeChkDesc(0, typeDescriptor)
	return typeDescriptor.m_descBuf, nil
}
