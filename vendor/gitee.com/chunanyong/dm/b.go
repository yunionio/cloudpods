/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

type ArrayDescriptor struct {
	m_typeDesc *TypeDescriptor
}

func newArrayDescriptor(fulName string, conn *DmConnection) (*ArrayDescriptor, error) {

	ad := new(ArrayDescriptor)

	if fulName == "" {
		return nil, ECGO_INVALID_COMPLEX_TYPE_NAME.throw()
	}

	ad.m_typeDesc = newTypeDescriptorWithFulName(fulName, conn)
	err := ad.m_typeDesc.parseDescByName()
	if err != nil {
		return nil, err
	}

	return ad, nil
}

func newArrayDescriptorByTypeDescriptor(desc *TypeDescriptor) *ArrayDescriptor {
	ad := new(ArrayDescriptor)
	ad.m_typeDesc = desc
	return ad
}

func (ad *ArrayDescriptor) getMDesc() *TypeDescriptor {
	return ad.m_typeDesc
}

func (ad *ArrayDescriptor) getItemDesc() *TypeDescriptor {
	return ad.m_typeDesc.m_arrObj
}

func (ad *ArrayDescriptor) getLength() int {
	return ad.m_typeDesc.m_length
}
