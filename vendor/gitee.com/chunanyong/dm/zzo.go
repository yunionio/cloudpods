/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"strconv"

	"gitee.com/chunanyong/dm/util"
)

const (
	ARRAY_TYPE_SHORT = 1

	ARRAY_TYPE_INTEGER = 2

	ARRAY_TYPE_LONG = 3

	ARRAY_TYPE_FLOAT = 4

	ARRAY_TYPE_DOUBLE = 5
)

var TypeDataSV TypeData

type InterfaceTypeData interface {
	toBytes(x *TypeData, typeDesc *TypeDescriptor) ([]byte, error)
}

type TypeData struct {
	m_dumyData interface{}

	m_offset int

	m_bufLen int

	m_dataBuf []byte

	m_objBlobDescBuf []byte

	m_isFromBlob bool

	m_packid int

	m_objRefArr []interface{}
}

func newTypeData(val interface{}, dataBuf []byte) *TypeData {
	td := new(TypeData).initTypeData()
	td.m_dumyData = val
	td.m_offset = 0
	td.m_bufLen = 0
	td.m_dataBuf = dataBuf
	return td
}

func (td *TypeData) initTypeData() *TypeData {
	td.m_dumyData = nil

	td.m_offset = 0

	td.m_bufLen = 0

	td.m_dataBuf = nil

	td.m_objBlobDescBuf = nil

	td.m_isFromBlob = false

	td.m_packid = -1

	td.m_objRefArr = make([]interface{}, 0)

	return td
}

func (sv TypeData) toStruct(objArr []interface{}, desc *TypeDescriptor) ([]TypeData, error) {
	size := desc.getStrctMemSize()
	retData := make([]TypeData, size)

	for i := 0; i < size; i++ {

		if objArr[i] == nil {
			retData[i] = *newTypeData(objArr[i], nil)
			continue
		}

		switch objArr[i].(type) {
		case DmStruct, DmArray:
			retData[i] = *newTypeData(objArr[i], nil)
		default:
			switch desc.m_fieldsObj[i].getDType() {
			case CLASS, PLTYPE_RECORD:
				tdArr, err := sv.toStruct(objArr[i].([]interface{}), &desc.m_fieldsObj[i])
				if err != nil {
					return nil, err
				}

				retData[i] = *newTypeData(newDmStructByTypeData(tdArr, &desc.m_fieldsObj[i]), nil)
			case ARRAY, SARRAY:
				tdArr, err := sv.toArray(objArr[i].([]interface{}), &desc.m_fieldsObj[i])
				if err != nil {
					return nil, err
				}

				retData[i] = *newTypeData(newDmArrayByTypeData(tdArr, &desc.m_fieldsObj[i]), nil)
			default:
				tdArr, err := sv.toMemberObj(objArr[i], &desc.m_fieldsObj[i])
				if err != nil {
					return nil, err
				}
				retData[i] = *tdArr
			}

		}
	}
	return retData, nil
}

func (sv TypeData) toArray(objArr []interface{}, desc *TypeDescriptor) ([]TypeData, error) {
	size := len(objArr)
	retData := make([]TypeData, size)
	for i := 0; i < size; i++ {
		if objArr[i] == nil {
			retData[i] = *newTypeData(objArr[i], nil)
			continue
		}

		switch objArr[i].(type) {
		case DmStruct, DmArray:
			retData[i] = *newTypeData(objArr[i], nil)
		default:
			switch desc.m_arrObj.getDType() {
			case CLASS, PLTYPE_RECORD:
				tdArr, err := sv.toStruct(objArr[i].([]interface{}), desc.m_arrObj)
				if err != nil {
					return nil, err
				}
				retData[i] = *newTypeData(newDmStructByTypeData(tdArr, desc.m_arrObj), nil)
			case ARRAY, SARRAY:

				tmp, ok := objArr[i].([]interface{})

				if !ok && desc.m_arrObj.m_arrObj != nil {
					obj, err := sv.makeupObjToArr(tmp[i], desc.m_arrObj)
					if err != nil {
						return nil, err
					}
					objArr[i] = obj
				}

				tdArr, err := sv.toArray(objArr[i].([]interface{}), desc.m_arrObj)
				if err != nil {
					return nil, err
				}

				retData[i] = *newTypeData(newDmArrayByTypeData(tdArr, desc.m_arrObj), nil)
			default:
				tdArr, err := sv.toMemberObj(objArr[i], desc.m_arrObj)
				if err != nil {
					return nil, err
				}
				retData[i] = *tdArr
			}
		}
	}

	return retData, nil
}

func (sv TypeData) makeupObjToArr(obj interface{}, objDesc *TypeDescriptor) ([]interface{}, error) {
	arrType := objDesc.getDType()
	dynamic := true
	arrLen := 0
	if arrType == SARRAY {
		dynamic = false
		arrLen = objDesc.m_length
	}

	subType := objDesc.m_arrObj.getDType()
	if subType == BINARY || subType == VARBINARY || subType == BIT {

		strRet := ""
		switch v := obj.(type) {
		case int:
			strRet = strconv.FormatInt(int64(v), 2)
		case int32:
			strRet = strconv.FormatInt(int64(v), 2)
		case int64:
			strRet = strconv.FormatInt(v, 2)
		case string:
			strRet = v
		default:
			return nil, ECGO_DATA_CONVERTION_ERROR.throw()
		}
		var prec int
		if dynamic {
			prec = len(strRet)
		} else {
			prec = arrLen
		}

		ret := make([]interface{}, prec)
		rs := Dm_build_650.Dm_build_866(strRet, objDesc.getServerEncoding(), objDesc.m_conn)
		for i := 0; i < prec; i++ {
			ret[i] = rs[i]
		}

		return ret, nil
	}

	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (sv TypeData) toMemberObj(mem interface{}, desc *TypeDescriptor) (*TypeData, error) {
	var bs []byte
	scale := desc.getScale()
	prec := desc.getPrec()
	dtype := desc.getDType()
	if mem == nil {
		return newTypeData(nil, nil), nil
	}

	param := new(parameter).InitParameter()
	param.colType = int32(dtype)
	param.prec = int32(prec)
	param.scale = int32(scale)

	var err error
	bs, err = G2DB.fromObject(mem, *param, desc.m_conn)
	if err != nil {
		return nil, err
	}

	return newTypeData(mem, bs), nil
}

func (sv TypeData) typeDataToBytes(data *TypeData, desc *TypeDescriptor) ([]byte, error) {
	dType := desc.getDType()
	var innerData []byte
	var err error
	if nil == data.m_dumyData {
		innerData = sv.realocBuffer(nil, 0, 2)
		Dm_build_650.Dm_build_651(innerData, 0, byte(0))
		Dm_build_650.Dm_build_651(innerData, 1, byte(0))
		return innerData, nil
	}

	var result []byte
	var offset int
	switch dType {
	case ARRAY:

		innerData, err = sv.arrayToBytes(data.m_dumyData.(*DmArray), desc)
		if err != nil {
			return nil, err
		}

		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1
		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])
		return result, nil

	case SARRAY:

		innerData, err = sv.sarrayToBytes(data.m_dumyData.(*DmArray), desc)
		if err != nil {
			return nil, err
		}
		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1

		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])
		return result, nil

	case CLASS:

		innerData, err = sv.objToBytes(data.m_dumyData, desc)
		if err != nil {
			return nil, err
		}
		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1
		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])
		return result, nil

	case PLTYPE_RECORD:

		innerData, err = sv.recordToBytes(data.m_dumyData.(*DmStruct), desc)
		if err != nil {
			return nil, err
		}
		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1

		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])
		return result, nil

	case BLOB, CLOB:
		innerData, err = sv.convertLobToBytes(data.m_dumyData, int(desc.column.colType), desc.getServerEncoding())

		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1
		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])
		return result, nil

	case BOOLEAN:
		innerData = sv.realocBuffer(nil, 0, 2)
		Dm_build_650.Dm_build_651(innerData, 0, byte(0))
		if data.m_dataBuf != nil && len(data.m_dataBuf) > 0 {
			Dm_build_650.Dm_build_651(innerData, 1, data.m_dataBuf[0])
		} else {
			Dm_build_650.Dm_build_651(innerData, 1, byte(0))
		}
		return innerData, nil

	default:

		innerData = data.m_dataBuf
		result = sv.realocBuffer(nil, 0, len(innerData)+BYTE_SIZE+BYTE_SIZE+USINT_SIZE)

		Dm_build_650.Dm_build_651(result, 0, byte(0))
		offset = 1

		Dm_build_650.Dm_build_651(result, offset, byte(1))
		offset += 1

		Dm_build_650.Dm_build_661(result, offset, int16(len(innerData)))
		offset += 2

		copy(result[offset:offset+len(innerData)], innerData[:len(innerData)])

		return result, nil
	}
}

func (sv TypeData) convertLobToBytes(value interface{}, dtype int, serverEncoding string) ([]byte, error) {
	var tmp []byte
	var ret []byte
	if dtype == BLOB {
		lob, ok := value.(DmBlob)
		if ok {
			l, err := lob.GetLength()
			if err != nil {
				return nil, err
			}
			tmp, err = lob.getBytes(1, int32(l))
			if err != nil {
				return nil, err
			}

			ret = make([]byte, l+ULINT_SIZE)
			Dm_build_650.Dm_build_666(ret, 0, int32(l))
			copy(ret[:ULINT_SIZE:ULINT_SIZE+l], tmp[:l])
			return ret, nil
		}

	}

	if dtype == CLOB {
		lob, ok := value.(DmClob)
		if ok {
			l, err := lob.GetLength()
			if err != nil {
				return nil, err
			}

			subString, err := lob.getSubString(1, int32(l))
			if err != nil {
				return nil, err
			}

			tmp = Dm_build_650.Dm_build_866(subString, serverEncoding, nil)
			ret = make([]byte, len(tmp)+ULINT_SIZE)
			Dm_build_650.Dm_build_666(ret, 0, int32(l))
			copy(ret[:ULINT_SIZE:ULINT_SIZE+l], tmp[:l])
		}
		return ret, nil
	}

	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (sv TypeData) sarrayToBytes(data *DmArray, desc *TypeDescriptor) ([]byte, error) {
	realLen := len(data.m_arrData)
	results := make([][]byte, realLen)
	var rdata []byte
	var err error

	if desc.getObjId() == 4 {
		return sv.ctlnToBytes(data, desc)
	}

	totalLen := 0
	for i := 0; i < realLen; i++ {
		results[i], err = sv.typeDataToBytes(&data.m_arrData[i], desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		totalLen += len(results[i])
	}

	totalLen += (ULINT_SIZE + ULINT_SIZE)
	rdata = sv.realocBuffer(nil, 0, totalLen)
	off := 0

	Dm_build_650.Dm_build_666(rdata, off, int32(totalLen))
	off += ULINT_SIZE

	Dm_build_650.Dm_build_666(rdata, off, int32(data.m_arrDesc.getLength()))
	off += ULINT_SIZE

	for i := 0; i < realLen; i++ {
		copy(rdata[off:off+len(results[i])], results[i][:len(results[i])])
		off += len(results[i])
	}

	return rdata, nil
}

func (sv TypeData) ctlnToBytes(data *DmArray, desc *TypeDescriptor) ([]byte, error) {
	results := make([][]byte, len(data.m_arrData))
	var rdata []byte
	var err error

	var totalLen int
	totalLen = BYTE_SIZE + ULINT_SIZE

	totalLen += USINT_SIZE + USINT_SIZE + ULINT_SIZE

	for i := 0; i < len(data.m_arrData); i++ {
		results[i], err = sv.typeDataToBytes(&data.m_arrData[i], desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		totalLen += len(results[i])
	}

	rdata = sv.realocBuffer(nil, 0, totalLen)

	offset := 0

	Dm_build_650.Dm_build_651(rdata, offset, byte(0))
	offset += BYTE_SIZE

	offset += ULINT_SIZE

	Dm_build_650.Dm_build_661(rdata, offset, int16(desc.getCltnType()))
	offset += USINT_SIZE

	Dm_build_650.Dm_build_661(rdata, offset, int16(desc.m_arrObj.getDType()))
	offset += USINT_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, int32(len(data.m_arrData)))
	offset += ULINT_SIZE

	for i := 0; i < len(data.m_arrData); i++ {
		copy(rdata[offset:offset+len(results[i])], results[i][:len(results[i])])
		offset += len(results[i])
	}

	Dm_build_650.Dm_build_666(rdata, BYTE_SIZE, int32(offset))

	return rdata, nil
}

func (sv TypeData) arrayToBytes(data *DmArray, desc *TypeDescriptor) ([]byte, error) {
	results := make([][]byte, len(data.m_arrData))
	var rdata []byte
	var err error
	if desc.getObjId() == 4 {
		return sv.ctlnToBytes(data, desc)
	}

	totalLen := 0
	for i := 0; i < len(data.m_arrData); i++ {
		results[i], err = sv.typeDataToBytes(&data.m_arrData[i], desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		totalLen += len(results[i])
	}

	totalLen += (ULINT_SIZE + ULINT_SIZE + ULINT_SIZE + ULINT_SIZE + ULINT_SIZE)

	total := data.m_objCount + data.m_strCount
	if total > 0 {
		totalLen += USINT_SIZE * total
	}

	rdata = sv.realocBuffer(nil, 0, totalLen)

	Dm_build_650.Dm_build_666(rdata, 0, int32(totalLen))
	offset := ULINT_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, int32(len(data.m_arrData)))
	offset += ULINT_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, 0)
	offset += ULINT_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, int32(data.m_objCount))
	offset += ULINT_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, int32(data.m_strCount))
	offset += ULINT_SIZE

	for i := 0; i < total; i++ {
		Dm_build_650.Dm_build_666(rdata, offset, int32(data.m_objStrOffs[i]))
		offset += ULINT_SIZE
	}

	for i := 0; i < len(data.m_arrData); i++ {
		copy(rdata[offset:offset+len(results[i])], results[i][:len(results[i])])
		offset += len(results[i])
	}

	return rdata, nil
}

func (sv TypeData) objToBytes(data interface{}, desc *TypeDescriptor) ([]byte, error) {

	switch data.(type) {
	case *DmArray:
		return sv.arrayToBytes(data.(*DmArray), desc)
	default:
		return sv.structToBytes(data.(*DmStruct), desc)
	}
}

func (sv TypeData) structToBytes(data *DmStruct, desc *TypeDescriptor) ([]byte, error) {
	size := desc.getStrctMemSize()
	results := make([][]byte, size)
	var rdata []byte
	var err error

	totalLen := 0
	for i := 0; i < size; i++ {
		results[i], err = sv.typeDataToBytes(&data.m_attribs[i], &desc.m_fieldsObj[i])
		if err != nil {
			return nil, err
		}
		totalLen += len(results[i])
	}

	totalLen += (BYTE_SIZE + ULINT_SIZE)

	rdata = sv.realocBuffer(nil, 0, totalLen)
	offset := 0

	Dm_build_650.Dm_build_651(rdata, offset, byte(0))
	offset += BYTE_SIZE

	Dm_build_650.Dm_build_666(rdata, offset, int32(totalLen))
	offset += ULINT_SIZE

	for i := 0; i < size; i++ {
		copy(rdata[offset:offset+len(results[i])], results[i][:len(results[i])])
		offset += len(results[i])
	}

	return rdata, nil
}

func (sv TypeData) recordToBytes(data *DmStruct, desc *TypeDescriptor) ([]byte, error) {
	size := desc.getStrctMemSize()
	results := make([][]byte, size)
	var rdata []byte
	var err error

	totalLen := 0
	for i := 0; i < size; i++ {
		results[i], err = sv.typeDataToBytes(&data.m_attribs[i], &desc.m_fieldsObj[i])
		if err != nil {
			return nil, err
		}
		totalLen += len(results[i])
	}

	totalLen += ULINT_SIZE
	rdata = sv.realocBuffer(nil, 0, totalLen)
	Dm_build_650.Dm_build_666(rdata, 0, int32(totalLen))

	offset := ULINT_SIZE
	for i := 0; i < desc.getStrctMemSize(); i++ {
		copy(rdata[offset:offset+len(results[i])], results[i][:len(results[i])])
		offset += len(results[i])
	}

	return rdata, nil
}

func (sv TypeData) bytesToBlob(val []byte, out *TypeData, desc *TypeDescriptor) (*TypeData, error) {
	offset := out.m_offset
	l := Dm_build_650.Dm_build_752(val, offset)
	offset += ULINT_SIZE

	tmp := Dm_build_650.Dm_build_801(val, offset, int(l))
	offset += int(l)
	out.m_offset = offset

	return newTypeData(newBlobOfLocal(tmp, desc.m_conn), tmp), nil
}

func (sv TypeData) bytesToClob(val []byte, out *TypeData, desc *TypeDescriptor, serverEncoding string) (*TypeData, error) {
	offset := out.m_offset
	l := Dm_build_650.Dm_build_752(val, offset)
	offset += ULINT_SIZE

	tmp := Dm_build_650.Dm_build_801(val, offset, int(l))
	offset += int(l)
	out.m_offset = offset

	return newTypeData(newClobOfLocal(Dm_build_650.Dm_build_807(tmp, 0, len(tmp), serverEncoding, desc.m_conn), desc.m_conn), tmp), nil
}

func (sv TypeData) bytesToTypeData(val []byte, out *TypeData, desc *TypeDescriptor) (*TypeData, error) {
	offset := out.m_offset

	offset += 1

	null_flag := Dm_build_650.Dm_build_743(val, offset)
	offset += 1

	out.m_offset = offset

	if desc.getDType() == BOOLEAN {
		b := false
		if null_flag != byte(0) {
			b = true
		}

		tmp := Dm_build_650.Dm_build_801(val, offset-1, 1)
		return newTypeData(b, tmp), nil
	}

	var retObj interface{}
	var err error
	var retDataBuf []byte
	switch desc.getDType() {
	case CLASS:
		if null_flag&byte(1) != byte(0) {
			retObj, err = sv.bytesToObj(val, out, desc)
			if err != nil {
				return nil, err
			}

			if out.m_offset > offset {
				retDataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)
			}

			return newTypeData(retObj, retDataBuf), nil
		} else {
			return newTypeData(nil, nil), nil
		}

	case ARRAY:
		if (null_flag & byte(1)) != byte(0) {
			retObj, err = sv.bytesToArray(val, out, desc)
			if err != nil {
				return nil, err
			}

			if out.m_offset > offset {
				retDataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)
			}

			return newTypeData(retObj, retDataBuf), nil
		} else {
			return newTypeData(nil, nil), nil
		}

	case PLTYPE_RECORD:
		if (null_flag & byte(1)) != byte(0) {
			retObj, err = sv.bytesToRecord(val, out, desc)
			if err != nil {
				return nil, err
			}

			if out.m_offset > offset {
				retDataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)
			}

			return newTypeData(retObj, retDataBuf), nil
		} else {
			return newTypeData(nil, nil), nil
		}

	case SARRAY:
		if (null_flag & byte(1)) != byte(0) {
			retObj, err = sv.bytesToSArray(val, out, desc)
			if err != nil {
				return nil, err
			}

			if out.m_offset > offset {
				retDataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)
			}

			return newTypeData(retObj, retDataBuf), nil
		} else {
			return newTypeData(nil, nil), nil
		}

	case BLOB:
		if null_flag&byte(1) != byte(0) {
			return sv.bytesToBlob(val, out, desc)
		} else {
			return newTypeData(nil, nil), nil
		}

	case CLOB:
		if null_flag&byte(1) != byte(0) {
			return sv.bytesToClob(val, out, desc, desc.getServerEncoding())
		} else {
			return newTypeData(nil, nil), nil
		}

	default:
		if null_flag&byte(1) != byte(0) {
			return sv.convertBytes2BaseData(val, out, desc)
		} else {
			return newTypeData(nil, nil), nil
		}

	}
}

func (sv TypeData) checkObjExist(val []byte, out *TypeData) bool {
	offset := out.m_offset
	exist_flag := Dm_build_650.Dm_build_743(val, offset)
	offset += 1

	out.m_offset = offset

	if exist_flag == byte(1) {
		return true
	}

	out.m_offset += ULINT_SIZE
	return false
}

func (sv TypeData) findObjByPackId(val []byte, out *TypeData) (*DmStruct, error) {
	offset := out.m_offset

	pack_id := int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	out.m_offset = offset

	if pack_id < 0 || pack_id > out.m_packid {
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}

	return out.m_objRefArr[pack_id].(*DmStruct), nil
}

func (sv TypeData) addObjToRefArr(out *TypeData, objToAdd interface{}) {
	out.m_objRefArr = append(out.m_objRefArr, objToAdd)
	out.m_packid++
}

func (sv TypeData) checkObjClnt(desc *TypeDescriptor) bool {
	return desc.m_objId == 4
}

func (sv TypeData) bytesToObj_EXACT(val []byte, out *TypeData, desc *TypeDescriptor) (*DmStruct, error) {
	strOut := newDmStructByTypeData(nil, desc)
	var sub_desc *TypeDescriptor
	offset := out.m_offset

	size := desc.getStrctMemSize()

	out.m_offset = offset

	strOut.m_attribs = make([]TypeData, size)
	for i := 0; i < size; i++ {
		sub_desc = &desc.m_fieldsObj[i]
		tmp, err := sv.bytesToTypeData(val, out, sub_desc)
		if err != nil {
			return nil, err
		}
		strOut.m_attribs[i] = *tmp
	}

	strOut.m_dataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)

	return strOut, nil
}

func (sv TypeData) bytesToNestTab(val []byte, out *TypeData, desc *TypeDescriptor) (*DmArray, error) {
	offset := out.m_offset

	offset += USINT_SIZE

	count := Dm_build_650.Dm_build_752(val, offset)
	offset += ULINT_SIZE

	out.m_offset = offset

	arrOut := newDmArrayByTypeData(nil, desc)
	arrOut.m_itemCount = int(count)
	arrOut.m_arrData = make([]TypeData, count)
	for i := 0; i < int(count); i++ {
		tmp, err := sv.bytesToTypeData(val, out, desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		arrOut.m_arrData[i] = *tmp
	}

	arrOut.m_dataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)

	return arrOut, nil
}

func (sv TypeData) bytesToClnt(val []byte, out *TypeData, desc *TypeDescriptor) (*DmArray, error) {
	var array *DmArray

	offset := out.m_offset

	cltn_type := Dm_build_650.Dm_build_747(val, offset)
	offset += USINT_SIZE

	out.m_offset = offset
	switch cltn_type {
	case CLTN_TYPE_IND_TABLE:
		return nil, ECGO_UNSUPPORTED_TYPE.throw()

	case CLTN_TYPE_NST_TABLE, CLTN_TYPE_VARRAY:
		return sv.bytesToNestTab(val, out, desc)
	}

	return array, nil
}

func (sv TypeData) bytesToObj(val []byte, out *TypeData, desc *TypeDescriptor) (interface{}, error) {
	var retObj interface{}
	var err error
	if out == nil {
		out = newTypeData(nil, nil)
	}

	if sv.checkObjExist(val, out) {
		retObj, err = sv.findObjByPackId(val, out)
		if err != nil {
			return nil, err
		}
	} else {
		sv.addObjToRefArr(out, retObj)
	}

	if sv.checkObjClnt(desc) {
		retObj, err = sv.bytesToClnt(val, out, desc)
		if err != nil {
			return nil, err
		}
	} else {
		retObj, err = sv.bytesToObj_EXACT(val, out, desc)
		if err != nil {
			return nil, err
		}
	}

	return retObj, nil
}

func (sv TypeData) bytesToArray(val []byte, out *TypeData, desc *TypeDescriptor) (*DmArray, error) {
	arrOut := newDmArrayByTypeData(nil, desc)
	if out == nil {
		out = newTypeData(nil, nil)
	}

	offset := out.m_offset

	arrOut.m_bufLen = int(Dm_build_650.Dm_build_752(val, offset))
	offset += 4

	arrOut.m_itemCount = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	arrOut.m_itemSize = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	arrOut.m_objCount = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	arrOut.m_strCount = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	total := arrOut.m_objCount + arrOut.m_strCount
	arrOut.m_objStrOffs = make([]int, total)
	for i := 0; i < total; i++ {
		arrOut.m_objStrOffs[i] = int(Dm_build_650.Dm_build_752(val, offset))
		offset += 4
	}

	out.m_offset = offset

	arrOut.m_arrData = make([]TypeData, arrOut.m_itemCount)
	for i := 0; i < arrOut.m_itemCount; i++ {
		tmp, err := sv.bytesToTypeData(val, out, desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		arrOut.m_arrData[i] = *tmp
	}

	arrOut.m_dataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)

	return arrOut, nil
}

func (sv TypeData) bytesToSArray(val []byte, out *TypeData, desc *TypeDescriptor) (*DmArray, error) {
	if out == nil {
		out = newTypeData(nil, nil)
	}

	offset := out.m_offset

	arrOut := newDmArrayByTypeData(nil, desc)
	arrOut.m_bufLen = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	arrOut.m_itemCount = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	out.m_offset = offset

	arrOut.m_arrData = make([]TypeData, arrOut.m_itemCount)
	for i := 0; i < arrOut.m_itemCount; i++ {
		tmp, err := sv.bytesToTypeData(val, out, desc.m_arrObj)
		if err != nil {
			return nil, err
		}
		arrOut.m_arrData[i] = *tmp
	}

	arrOut.m_dataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)

	return arrOut, nil
}

func (sv TypeData) bytesToRecord(val []byte, out *TypeData, desc *TypeDescriptor) (*DmStruct, error) {
	if out == nil {
		out = newTypeData(nil, nil)
	}

	offset := out.m_offset

	strOut := newDmStructByTypeData(nil, desc)
	strOut.m_bufLen = int(Dm_build_650.Dm_build_752(val, offset))
	offset += ULINT_SIZE

	out.m_offset = offset

	strOut.m_attribs = make([]TypeData, desc.getStrctMemSize())
	for i := 0; i < desc.getStrctMemSize(); i++ {
		tmp, err := sv.bytesToTypeData(val, out, &desc.m_fieldsObj[i])
		if err != nil {
			return nil, err
		}
		strOut.m_attribs[i] = *tmp
	}

	strOut.m_dataBuf = Dm_build_650.Dm_build_801(val, offset, out.m_offset-offset)

	return strOut, nil
}

func (sv TypeData) objBlob_GetChkBuf(buf []byte, typeData *TypeData) {

	offset := 4

	l := int(Dm_build_650.Dm_build_752(buf, offset))
	offset += ULINT_SIZE

	typeData.m_objBlobDescBuf = Dm_build_650.Dm_build_801(buf, offset, l)
	offset += l

	typeData.m_isFromBlob = true

	typeData.m_offset = offset
}

func (sv TypeData) objBlobToObj(lob *DmBlob, desc *TypeDescriptor) (interface{}, error) {
	typeData := newTypeData(nil, nil)
	loblen, err := lob.GetLength()
	if err != nil {
		return nil, err
	}

	buf, err := lob.getBytes(1, int32(loblen))
	if err != nil {
		return nil, err
	}

	sv.objBlob_GetChkBuf(buf, typeData)

	return sv.bytesToObj(buf, typeData, desc)
}

func (sv TypeData) objBlobToBytes(lobBuf []byte, desc *TypeDescriptor) ([]byte, error) {
	l := len(lobBuf)
	offset := 0

	magic := Dm_build_650.Dm_build_752(lobBuf, offset)
	offset += ULINT_SIZE

	if OBJ_BLOB_MAGIC != magic {
		return nil, ECGO_INVALID_OBJ_BLOB.throw()
	}

	descLen := int(Dm_build_650.Dm_build_752(lobBuf, offset))
	offset += ULINT_SIZE
	descBuf := Dm_build_650.Dm_build_801(lobBuf, offset, descLen)
	tmp, err := desc.getClassDescChkInfo()
	if err != nil {
		return nil, err
	}
	if !util.SliceEquals(descBuf, tmp) {
		return nil, ECGO_INVALID_OBJ_BLOB.throw()
	}
	offset += descLen

	ret := make([]byte, l-offset)
	copy(ret[:len(ret)], lobBuf[offset:offset+len(ret)])
	return ret, nil
}

func (sv TypeData) realocBuffer(oldBuf []byte, offset int, needLen int) []byte {
	var retBuf []byte

	if oldBuf == nil {
		return make([]byte, needLen)
	} else if needLen+offset > len(oldBuf) {
		retBuf = make([]byte, len(oldBuf)+needLen)
		copy(retBuf[:offset], oldBuf[:offset])
	} else {
		retBuf = oldBuf
	}

	return retBuf
}

func (sv TypeData) convertBytes2BaseData(val []byte, out *TypeData, desc *TypeDescriptor) (*TypeData, error) {
	offset := out.m_offset
	isNull := false
	valueLen := int(Dm_build_650.Dm_build_774(val, offset))
	offset += USINT_SIZE

	if valueLen == int(Dm_build_60) {
		valueLen = 0
		isNull = true
	}

	if -1 == valueLen {
		valueLen = int(Dm_build_650.Dm_build_752(val, offset))
		offset += ULINT_SIZE
	}

	if isNull {
		out.m_offset = offset
		return newTypeData(nil, nil), nil
	}

	var tmpObj interface{}
	var err error
	temp := Dm_build_650.Dm_build_801(val, offset, valueLen)
	offset += valueLen
	out.m_offset = offset

	tmpObj, err = DB2G.toObject(temp, desc.column, desc.m_conn)
	if err != nil {
		return nil, err
	}
	return newTypeData(tmpObj, temp), nil
}

func (td *TypeData) toJavaArray(arr *DmArray, index int64, l int, dType int) (interface{}, error) {
	if arr.m_objArray != nil {
		return arr.m_objArray, nil
	}

	var nr = make([]interface{}, l)
	var tempData *TypeData
	switch dType {
	case CHAR, VARCHAR, VARCHAR2:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}
	case BIT, TINYINT:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case BINARY, VARBINARY:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case BOOLEAN:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case SMALLINT:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case INT:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case BIGINT:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case DECIMAL:

		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}
	case REAL:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case DOUBLE:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			} else {
				nr[i] = nil
			}
		}

	case DATE, TIME, DATETIME, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case INTERVAL_YM:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case INTERVAL_DT:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case PLTYPE_RECORD, CLASS, ARRAY, SARRAY:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case BLOB:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	case CLOB:
		for i := 0; i < l; i++ {
			tempData = &arr.m_arrData[index+int64(i)]
			if tempData != nil && tempData.m_dumyData != nil {
				nr[i] = tempData.m_dumyData
			}
		}

	default:
		return nil, ECGO_UNSUPPORTED_TYPE.throw()
	}

	return nr, nil
}

func (td *TypeData) toNumericArray(arr *DmArray, index int64, l int, flag int) (interface{}, error) {
	if nil == arr.m_objArray {
		return nil, nil
	}

	var retObj interface{}
	switch arr.m_objArray.(type) {
	case []int16:
		if flag == ARRAY_TYPE_SHORT {
			ret := make([]int16, l)
			copy(ret[:l], arr.m_objArray.([]int16)[index:index+int64(l)])
			retObj = ret
		}
	case []int:
		if flag == ARRAY_TYPE_INTEGER {
			ret := make([]int, l)
			copy(ret[:l], arr.m_objArray.([]int)[index:index+int64(l)])
			retObj = ret
		}
	case []int64:
		if flag == ARRAY_TYPE_LONG {
			ret := make([]int64, l)
			copy(ret[:l], arr.m_objArray.([]int64)[index:index+int64(l)])
			retObj = ret
		}
	case []float32:
		if flag == ARRAY_TYPE_FLOAT {
			ret := make([]float32, l)
			copy(ret[:l], arr.m_objArray.([]float32)[index:index+int64(l)])
			retObj = ret
		}
	case []float64:
		if flag == ARRAY_TYPE_DOUBLE {
			ret := make([]float64, l)
			copy(ret[:l], arr.m_objArray.([]float64)[index:index+int64(l)])
			retObj = ret
		}
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}

	return retObj, nil
}

func (td *TypeData) toJavaArrayByDmStruct(st *DmStruct) ([]interface{}, error) {
	attrsData := st.getAttribsTypeData()
	if nil == st.getAttribsTypeData() || len(st.getAttribsTypeData()) <= 0 {
		return nil, nil
	}

	fields := st.m_strctDesc.getItemsDesc()
	if len(attrsData) != len(fields) {
		return nil, ECGO_STRUCT_MEM_NOT_MATCH.throw()
	}

	out := make([]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		out[i] = attrsData[i].m_dumyData
	}

	return out, nil
}

func (td *TypeData) toBytesFromDmArray(x *DmArray, typeDesc *TypeDescriptor) ([]byte, error) {
	var err error
	desc, err := typeDesc.getClassDescChkInfo()
	if err != nil {
		return nil, err
	}
	var data []byte
	switch typeDesc.getDType() {
	case ARRAY:
		data, err = TypeDataSV.arrayToBytes(x, typeDesc)
		if err != nil {
			return nil, err
		}
	case SARRAY:
		data, err = TypeDataSV.sarrayToBytes(x, typeDesc)
		if err != nil {
			return nil, err
		}
	case PLTYPE_RECORD:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	case CLASS:
		data, err = TypeDataSV.objToBytes(x, typeDesc)
		if err != nil {
			return nil, err
		}
	}
	ret := make([]byte, ULINT_SIZE+ULINT_SIZE+len(desc)+len(data))
	Dm_build_650.Dm_build_666(ret, 0, OBJ_BLOB_MAGIC)
	Dm_build_650.Dm_build_666(ret, ULINT_SIZE, int32(len(desc)))
	copy(ret[ULINT_SIZE+ULINT_SIZE:ULINT_SIZE+ULINT_SIZE+len(desc)], desc[:len(desc)])
	copy(ret[ULINT_SIZE+ULINT_SIZE+len(desc):ULINT_SIZE+ULINT_SIZE+len(desc)+len(data)], data[:len(data)])
	return ret, nil
}

func (td *TypeData) toBytesFromDmStruct(x *DmStruct, typeDesc *TypeDescriptor) ([]byte, error) {
	var err error
	desc, err := typeDesc.getClassDescChkInfo()
	if err != nil {
		return nil, err
	}
	var data []byte
	switch typeDesc.getDType() {
	case ARRAY:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	case SARRAY:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	case PLTYPE_RECORD:
		data, err = TypeDataSV.recordToBytes(x, typeDesc)
		if err != nil {
			return nil, err
		}
	case CLASS:
		data, err = TypeDataSV.objToBytes(x, typeDesc)
		if err != nil {
			return nil, err
		}
	}
	ret := make([]byte, ULINT_SIZE+ULINT_SIZE+len(desc)+len(data))
	Dm_build_650.Dm_build_666(ret, 0, OBJ_BLOB_MAGIC)
	Dm_build_650.Dm_build_666(ret, ULINT_SIZE, int32(len(desc)))
	copy(ret[ULINT_SIZE+ULINT_SIZE:ULINT_SIZE+ULINT_SIZE+len(desc)], desc[:len(desc)])
	copy(ret[ULINT_SIZE+ULINT_SIZE+len(desc):ULINT_SIZE+ULINT_SIZE+len(desc)+len(data)], data[:len(data)])
	return ret, nil
}
