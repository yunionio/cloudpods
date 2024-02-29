/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

type StructDescriptor struct {
	m_typeDesc *TypeDescriptor
}

func newStructDescriptor(fulName string, conn *DmConnection) (*StructDescriptor, error) {
	sd := new(StructDescriptor)
	if fulName == "" {
		return nil, ECGO_INVALID_COMPLEX_TYPE_NAME.throw()
	}

	sd.m_typeDesc = newTypeDescriptorWithFulName(fulName, conn)

	err := sd.m_typeDesc.parseDescByName()
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func newStructDescriptorByTypeDescriptor(desc *TypeDescriptor) *StructDescriptor {
	sd := new(StructDescriptor)
	sd.m_typeDesc = desc
	return sd
}

func (sd *StructDescriptor) getSize() int {
	return sd.m_typeDesc.m_size
}

func (sd *StructDescriptor) getObjId() int {
	return sd.m_typeDesc.m_objId
}

func (sd *StructDescriptor) getItemsDesc() []TypeDescriptor {
	return sd.m_typeDesc.m_fieldsObj
}
