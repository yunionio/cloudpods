package jsonutils

func Update(dst, src interface{}) error {
	json := Marshal(src)
	return json.Unmarshal(dst)
}
