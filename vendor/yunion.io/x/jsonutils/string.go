package jsonutils

import (
	"fmt"
	"strings"
)

func (this *JSONString) String() string {
	return quoteString(this.data)
}

func (this *JSONValue) String() string {
	return "null"
}

func (this *JSONInt) String() string {
	return fmt.Sprintf("%d", this.data)
}

func (this *JSONFloat) String() string {
	return fmt.Sprintf("%f", this.data)
}

func (this *JSONBool) String() string {
	if this.data {
		return "true"
	} else {
		return "false"
	}
}

func (this *JSONDict) String() string {
	sb := &strings.Builder{}
	this.buildString(sb)
	return sb.String()
}

func (this *JSONArray) String() string {
	sb := &strings.Builder{}
	this.buildString(sb)
	return sb.String()
}
