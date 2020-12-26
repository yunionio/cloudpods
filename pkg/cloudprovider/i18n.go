package cloudprovider

import (
	"yunion.io/x/onecloud/pkg/i18n"
)

type SModelI18nEntry struct {
	Value     string
	valueI18n map[i18n.Tag]string
}

func NewSModelI18nEntry(value string) *SModelI18nEntry {
	vn := make(map[i18n.Tag]string, 0)
	return &SModelI18nEntry{Value: value, valueI18n: vn}
}

func (self *SModelI18nEntry) GetKeyValue() string {
	return self.Value
}

func (self *SModelI18nEntry) Lookup(tag i18n.Tag) string {
	if v, ok := self.valueI18n[tag]; ok {
		return v
	}

	return self.Value
}

func (self *SModelI18nEntry) CN(v string) *SModelI18nEntry {
	self.valueI18n[i18n.I18N_TAG_CHINESE] = v
	return self
}

func (self *SModelI18nEntry) EN(v string) *SModelI18nEntry {
	self.valueI18n[i18n.I18N_TAG_ENGLISH] = v
	return self
}

type SModelI18nTable map[string]*SModelI18nEntry
