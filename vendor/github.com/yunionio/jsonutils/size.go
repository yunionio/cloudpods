package jsonutils

func (this *JSONDict) Size() int {
	return len(this.data)
}

func (this *JSONArray) Size() int {
	return len(this.data)
}
