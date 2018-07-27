package jsonutils

func (this *JSONValue) IsZero() bool {
	return true
}

func (this *JSONBool) IsZero() bool {
	return this.data == false
}

func (this *JSONInt) IsZero() bool {
	return this.data == 0
}

func (this *JSONFloat) IsZero() bool {
	return this.data == 0.0
}

func (this *JSONString) IsZero() bool {
	return len(this.data) == 0
}

func (this *JSONDict) IsZero() bool {
	return this.data == nil || len(this.data) == 0
}

func (this *JSONArray) IsZero() bool {
	return this.data == nil || len(this.data) == 0
}
