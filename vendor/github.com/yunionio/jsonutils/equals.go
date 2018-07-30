package jsonutils

// import "github.com/yunionio/pkg/gotypes"

func (dict *JSONDict) Equals(json JSONObject) bool {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return false
	}
	if len(dict.data) != len(dict2.data) {
		return false
	}
	for k, v := range dict.data {
		v2, ok := dict2.data[k]
		if !ok {
			return false
		}
		if !v.Equals(v2) {
			return false
		}
	}
	return true
}

func (arr *JSONArray) Equals(json JSONObject) bool {
	arr2, ok := json.(*JSONArray)
	if !ok {
		return false
	}
	if len(arr.data) != len(arr2.data) {
		return false
	}
	for i, v := range arr.data {
		if !v.Equals(arr2.data[i]) {
			return false
		}
	}
	return true
}

func (o *JSONString) Equals(json JSONObject) bool {
	o2, ok := json.(*JSONString)
	if !ok {
		return false
	}
	return o.data == o2.data
}

func (o *JSONInt) Equals(json JSONObject) bool {
	o2, ok := json.(*JSONInt)
	if !ok {
		return false
	}
	return o.data == o2.data
}

func (o *JSONFloat) Equals(json JSONObject) bool {
	o2, ok := json.(*JSONFloat)
	if !ok {
		return false
	}
	return o.data == o2.data
}

func (o *JSONBool) Equals(json JSONObject) bool {
	o2, ok := json.(*JSONBool)
	if !ok {
		return false
	}
	return o.data == o2.data
}

func (o *JSONValue) Equals(json JSONObject) bool {
	_, ok := json.(*JSONValue)
	if !ok {
		return false
	}
	return true
}
