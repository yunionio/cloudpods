package jsonutils

import (
	"strings"

	"yunion.io/x/pkg/sortedmap"
)

type writeSource interface {
	buildString(sb *strings.Builder)
}

func (this *JSONString) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONValue) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONInt) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONFloat) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONBool) buildString(sb *strings.Builder) {
	sb.WriteString(this.String())
}

func (this *JSONDict) buildString(sb *strings.Builder) {
	sb.WriteByte('{')
	var idx = 0
	for iter := sortedmap.NewIterator(this.data); iter.HasMore(); iter.Next() {
		k, vinf := iter.Get()
		v := vinf.(JSONObject)
		if idx > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(quoteString(k))
		sb.WriteByte(':')

		v.buildString(sb)
		idx++
	}
	sb.WriteByte('}')
}

func (this *JSONArray) buildString(sb *strings.Builder) {
	sb.WriteByte('[')
	for idx, v := range this.data {
		if idx > 0 {
			sb.WriteByte(',')
		}
		v.buildString(sb)
	}
	sb.WriteByte(']')
}
