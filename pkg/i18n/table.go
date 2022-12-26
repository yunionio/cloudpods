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

package i18n

import (
	"context"

	"golang.org/x/text/language"

	"yunion.io/x/pkg/appctx"
)

type Tag = language.Tag

var (
	I18N_TAG_CHINESE Tag = Tag(language.Chinese)
	I18N_TAG_ENGLISH Tag = Tag(language.English)
)

type ITable interface {
	Lookup(ctx context.Context, key string) string
}

type TableEntry map[language.Tag]string
type Table map[string]TableEntry

func NewTableEntry() TableEntry {
	return TableEntry{}
}

func (te TableEntry) CN(v string) TableEntry {
	te[language.Chinese] = v
	return te
}

func (te TableEntry) EN(v string) TableEntry {
	te[language.English] = v
	return te
}

func (te TableEntry) Lookup(ctx context.Context) (string, bool) {
	lang := appctx.Lang(ctx)
	lang = tableLangMatch(lang)

	v, ok := te[lang]
	return v, ok
}

func (tbl Table) Set(k string, te TableEntry) {
	tbl[k] = te
}

func (tbl Table) Lookup(ctx context.Context, key string) string {
	lang := appctx.Lang(ctx)
	return tbl.LookupByLang(lang, key)
}

func (tbl Table) LookupByLang(lang language.Tag, key string) string {
	te, ok := tbl[key]
	if !ok {
		return key
	}

	lang = tableLangMatch(lang)
	v, ok := te[lang]
	if !ok {
		return key
	}

	return v
}

var tableLangSupported = []language.Tag{
	language.English,
	language.Chinese,
}
var tableLangMatcher = language.NewMatcher(tableLangSupported)

func tableLangMatch(tag language.Tag) language.Tag {
	_, i, _ := tableLangMatcher.Match(tag)
	return tableLangSupported[i]
}
