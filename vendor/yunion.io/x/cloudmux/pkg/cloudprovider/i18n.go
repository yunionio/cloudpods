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
	"golang.org/x/text/language"
)

type SModelI18nEntry struct {
	Value     string
	valueI18n map[language.Tag]string
}

func NewSModelI18nEntry(value string) *SModelI18nEntry {
	vn := make(map[language.Tag]string, 0)
	return &SModelI18nEntry{Value: value, valueI18n: vn}
}

func (entry *SModelI18nEntry) GetKeyValue() string {
	return entry.Value
}

func (entry *SModelI18nEntry) Lookup(tag language.Tag) string {
	if v, ok := entry.valueI18n[tag]; ok {
		return v
	}
	return entry.Value
}

func (entry *SModelI18nEntry) CN(v string) *SModelI18nEntry {
	entry.valueI18n[language.Chinese] = v
	return entry
}

func (entry *SModelI18nEntry) EN(v string) *SModelI18nEntry {
	entry.valueI18n[language.English] = v
	return entry
}

type SModelI18nTable map[string]*SModelI18nEntry
