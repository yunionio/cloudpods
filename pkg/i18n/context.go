package i18n

import (
	"context"
	"net/http"

	"golang.org/x/text/language"
)

type ctxLang uintptr

const (
	ctxLangKey = ctxLang(0)
)

var (
	defaultLang = language.English
)

func WithLangTag(ctx context.Context, tag language.Tag) context.Context {
	return context.WithValue(ctx, ctxLangKey, tag)
}

func WithLang(ctx context.Context, lang string) context.Context {
	tag, err := language.Parse(lang)
	if err != nil {
		tag = defaultLang
	}
	return WithLangTag(ctx, tag)
}

func WithRequestLang(ctx context.Context, req *http.Request) context.Context {
	if val := req.URL.Query().Get("lang"); val != "" {
		return WithLang(ctx, val)
	}
	if val := req.Header.Get(LangHeader); val != "" {
		return WithLang(ctx, val)
	}
	if cookie, err := req.Cookie("lang"); err == nil {
		return WithLang(ctx, cookie.Value)
	}
	return WithLangTag(ctx, defaultLang)
}

func Lang(ctx context.Context) language.Tag {
	var (
		langv = ctx.Value(ctxLangKey)
		lang  language.Tag
	)
	if langv != nil {
		lang = langv.(language.Tag)
	} else {
		lang = defaultLang
	}
	return lang
}

const (
	LangHeader = "X-Yunion-Lang"
)

func SetHTTPLangHeader(ctx context.Context, header http.Header) bool {
	langv := ctx.Value(ctxLangKey)
	langTag, ok := langv.(language.Tag)
	if ok {
		header.Set(LangHeader, langTag.String())
	}
	return ok
}
