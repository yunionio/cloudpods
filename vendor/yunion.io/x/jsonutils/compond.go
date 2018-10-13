package jsonutils

func (val *JSONValue) isCompond() bool {
	return false
}

func (val *JSONDict) isCompond() bool {
	return true
}

func (val *JSONArray) isCompond() bool {
	return true
}
