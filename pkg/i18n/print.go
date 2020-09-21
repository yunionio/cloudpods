package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"yunion.io/x/onecloud/locales"
)

func P(tag language.Tag, id string, a ...interface{}) string {
	p := message.NewPrinter(tag, message.Catalog(locales.Catalog))
	return p.Sprintf(id, a...)
}
