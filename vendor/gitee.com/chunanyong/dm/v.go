/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import "database/sql/driver"

type DmStruct struct {
	TypeData
	m_strctDesc *StructDescriptor // 结构体的描述信息

	m_attribs []TypeData // 各属性值

	m_objCount int // 一个数组项中存在对象类型的个数（class、动态数组)

	m_strCount int // 一个数组项中存在字符串类型的个数

	typeName string

	elements []interface{}

	// Valid为false代表DmArray数据在数据库中为NULL
	Valid bool
}

// 数据库自定义类型Struct构造函数，typeName为库中定义的类型名称，elements为该类型每个字段的值
//
// 例如，自定义类型语句为：create or replace type myType as object (a1 int, a2 varchar);
//
// 则绑入绑出的go对象为: val := dm.NewDmStruct("myType", []interface{} {123, "abc"})
func NewDmStruct(typeName string, elements []interface{}) *DmStruct {
	ds := new(DmStruct)
	ds.typeName = typeName
	ds.elements = elements
	ds.Valid = true
	return ds
}

func (ds *DmStruct) create(dc *DmConnection) (*DmStruct, error) {
	desc, err := newStructDescriptor(ds.typeName, dc)
	if err != nil {
		return nil, err
	}
	return ds.createByStructDescriptor(desc, dc)
}

func newDmStructByTypeData(atData []TypeData, desc *TypeDescriptor) *DmStruct {
	ds := new(DmStruct)
	ds.Valid = true
	ds.initTypeData()
	ds.m_strctDesc = newStructDescriptorByTypeDescriptor(desc)
	ds.m_attribs = atData
	return ds
}

func (dest *DmStruct) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmStruct)
		// 将Valid标志置false表示数据库中该列为NULL
		(*dest).Valid = false
		return nil
	case *DmStruct:
		*dest = *src
		return nil
	default:
		return UNSUPPORTED_SCAN.throw()
	}
}

func (dt DmStruct) Value() (driver.Value, error) {
	if !dt.Valid {
		return nil, nil
	}
	return dt, nil
}

func (ds *DmStruct) getAttribsTypeData() []TypeData {
	return ds.m_attribs
}

func (ds *DmStruct) createByStructDescriptor(desc *StructDescriptor, conn *DmConnection) (*DmStruct, error) {
	ds.initTypeData()

	if nil == desc {
		return nil, ECGO_INVALID_PARAMETER_VALUE.throw()
	}

	ds.m_strctDesc = desc
	if nil == ds.elements {
		ds.m_attribs = make([]TypeData, desc.getSize())
	} else {
		if desc.getSize() != len(ds.elements) && desc.getObjId() != 4 {
			return nil, ECGO_STRUCT_MEM_NOT_MATCH.throw()
		}
		var err error
		ds.m_attribs, err = TypeDataSV.toStruct(ds.elements, ds.m_strctDesc.m_typeDesc)
		if err != nil {
			return nil, err
		}
	}

	return ds, nil
}

// 获取Struct对象在数据库中的类型名称
func (ds *DmStruct) GetSQLTypeName() (string, error) {
	return ds.m_strctDesc.m_typeDesc.getFulName()
}

// 获取Struct对象中的各个字段的值
func (ds *DmStruct) GetAttributes() ([]interface{}, error) {
	return TypeDataSV.toJavaArrayByDmStruct(ds)
}

func (ds *DmStruct) checkCol(col int) error {
	if col < 1 || col > len(ds.m_attribs) {
		return ECGO_INVALID_SEQUENCE_NUMBER.throw()
	}
	return nil
}

// 获取指定索引的成员变量值，以TypeData的形式给出，col 1 based
func (ds *DmStruct) getAttrValue(col int) (*TypeData, error) {
	err := ds.checkCol(col)
	if err != nil {
		return nil, err
	}
	return &ds.m_attribs[col-1], nil
}

func (ds *DmStruct) checkValid() error {
	if !ds.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}
