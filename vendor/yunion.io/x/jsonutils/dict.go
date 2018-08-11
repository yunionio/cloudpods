package jsonutils

func (dict *JSONDict) Update(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	for k, v := range dict2.data {
		dict.data[k] = v
	}
}

func (dict *JSONDict) UpdateDefault(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	for k, v := range dict2.data {
		if _, ok := dict.data[k]; !ok {
			dict.data[k] = v
		}
	}
}
