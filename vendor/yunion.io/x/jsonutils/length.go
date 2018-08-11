package jsonutils

func (this *JSONString) Length() int {
	return len(this.data)
}

func (this *JSONDict) Length() int {
	return len(this.data)
}

func (this *JSONArray) Length() int {
	return len(this.data)
}
