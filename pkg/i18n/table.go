package i18n

import (
	"context"

	"golang.org/x/text/language"
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

func (tbl Table) Set(k string, te TableEntry) {
	tbl[k] = te
}

func (tbl Table) Lookup(ctx context.Context, key string) string {
	te, ok := tbl[key]
	if !ok {
		return key
	}
	lang := Lang(ctx)
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
