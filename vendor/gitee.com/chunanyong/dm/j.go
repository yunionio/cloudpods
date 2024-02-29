/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import "database/sql/driver"

type DmArray struct {
	TypeData
	m_arrDesc *ArrayDescriptor // 数组的描述信息

	m_arrData []TypeData // 数组中各行数据值

	m_objArray interface{} // 从服务端获取的

	m_itemCount int // 本次获取的行数

	m_itemSize int // 数组中一个数组项的大小，单位bytes

	m_objCount int // 一个数组项中存在对象类型的个数（class、动态数组)

	m_strCount int // 一个数组项中存在字符串类型的个数

	m_objStrOffs []int // 对象在前，字符串在后

	typeName string

	elements []interface{}

	// Valid为false代表DmArray数据在数据库中为NULL
	Valid bool
}

func (da *DmArray) init() *DmArray {
	da.initTypeData()
	da.m_itemCount = 0
	da.m_itemSize = 0
	da.m_objCount = 0
	da.m_strCount = 0
	da.m_objStrOffs = nil
	da.m_dumyData = nil
	da.m_offset = 0

	da.m_objArray = nil
	da.Valid = true
	return da
}

// 数据库自定义数组Array构造函数，typeName为库中定义的数组类型名称，elements为该数组类型的每个值
//
// 例如，自定义数组类型语句为：create or replace type myArray is array int[];
//
// 则绑入绑出的go对象为: val := dm.NewDmArray("myArray", []interface{} {123, 456})
func NewDmArray(typeName string, elements []interface{}) *DmArray {
	da := new(DmArray)
	da.typeName = typeName
	da.elements = elements
	da.Valid = true
	return da
}

func (da *DmArray) create(dc *DmConnection) (*DmArray, error) {
	desc, err := newArrayDescriptor(da.typeName, dc)
	if err != nil {
		return nil, err
	}
	return da.createByArrayDescriptor(desc, dc)
}

func (da *DmArray) createByArrayDescriptor(arrDesc *ArrayDescriptor, conn *DmConnection) (*DmArray, error) {

	if nil == arrDesc {
		return nil, ECGO_INVALID_PARAMETER_VALUE.throw()
	}

	da.init()

	da.m_arrDesc = arrDesc
	if nil == da.elements {
		da.m_arrData = make([]TypeData, 0)
	} else {
		// 若为静态数组，判断给定数组长度是否超过静态数组的上限
		if arrDesc.getMDesc() == nil || (arrDesc.getMDesc().getDType() == SARRAY && len(da.elements) > arrDesc.getMDesc().getStaticArrayLength()) {
			return nil, ECGO_INVALID_ARRAY_LEN.throw()
		}

		var err error
		da.m_arrData, err = TypeDataSV.toArray(da.elements, da.m_arrDesc.getMDesc())
		if err != nil {
			return nil, err
		}
	}

	da.m_itemCount = len(da.m_arrData)
	return da, nil
}

func newDmArrayByTypeData(atData []TypeData, desc *TypeDescriptor) *DmArray {
	da := new(DmArray)
	da.init()
	da.m_arrDesc = newArrayDescriptorByTypeDescriptor(desc)
	da.m_arrData = atData
	return da
}

func (da *DmArray) checkIndex(index int64) error {
	if index < 0 || index > int64(len(da.m_arrData)-1) {
		return ECGO_INVALID_LENGTH_OR_OFFSET.throw()
	}
	return nil
}

func (da *DmArray) checkIndexAndCount(index int64, count int) error {
	err := da.checkIndex(index)
	if err != nil {
		return err
	}

	if count <= 0 || index+int64(count) > int64(len(da.m_arrData)) {
		return ECGO_INVALID_LENGTH_OR_OFFSET.throw()
	}
	return nil
}

// 获取Array对象在数据库中的类型名称
func (da *DmArray) GetBaseTypeName() (string, error) {
	if err := da.checkValid(); err != nil {
		return "", err
	}
	return da.m_arrDesc.m_typeDesc.getFulName()
}

// 获取Array对象的go数组对象
func (da *DmArray) GetArray() (interface{}, error) {
	if da.m_arrData == nil || len(da.m_arrData) <= 0 {
		return nil, nil;
	}

	return TypeDataSV.toJavaArray(da, 0, len(da.m_arrData), da.m_arrDesc.getItemDesc().getDType())
}

// 获取Array对象的指定偏移和执行长度go数据对象 index从0开始
func (da *DmArray) GetObjArray(index int64, count int) (interface{}, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	return TypeDataSV.toJavaArray(da, index, count, da.m_arrDesc.getItemDesc().getDType())
}

func (da *DmArray) GetIntArray(index int64, count int) ([]int, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	tmp, err := TypeDataSV.toNumericArray(da, index, count, ARRAY_TYPE_INTEGER)
	if err != nil {
		return nil, err
	}
	return tmp.([]int), nil
}

func (da *DmArray) GetInt16Array(index int64, count int) ([]int16, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	tmp, err := TypeDataSV.toNumericArray(da, index, count, ARRAY_TYPE_SHORT)
	if err != nil {
		return nil, err
	}
	return tmp.([]int16), nil
}

func (da *DmArray) GetInt64Array(index int64, count int) ([]int64, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	tmp, err := TypeDataSV.toNumericArray(da, index, count, ARRAY_TYPE_LONG)
	if err != nil {
		return nil, err
	}

	return tmp.([]int64), nil
}

func (da *DmArray) GetFloatArray(index int64, count int) ([]float32, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	tmp, err := TypeDataSV.toNumericArray(da, index, count, ARRAY_TYPE_FLOAT)
	if err != nil {
		return nil, err
	}

	return tmp.([]float32), nil
}

func (da *DmArray) GetDoubleArray(index int64, count int) ([]float64, error) {
	var err error
	if err = da.checkValid(); err != nil {
		return nil, err
	}
	if err = da.checkIndexAndCount(index, count); err != nil {
		return nil, err
	}

	tmp, err := TypeDataSV.toNumericArray(da, index, count, ARRAY_TYPE_DOUBLE)
	if err != nil {
		return nil, err
	}

	return tmp.([]float64), nil
}

func (dest *DmArray) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmArray)
		// 将Valid标志置false表示数据库中该列为NULL
		(*dest).Valid = false
		return nil
	case *DmArray:
		*dest = *src
		return nil
	default:
		return UNSUPPORTED_SCAN.throw()
	}
}

func (array DmArray) Value() (driver.Value, error) {
	if !array.Valid {
		return nil, nil
	}
	return array, nil
}

func (array *DmArray) checkValid() error {
	if !array.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}
