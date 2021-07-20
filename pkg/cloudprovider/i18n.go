// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
