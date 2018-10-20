package jsonutils

func (self *JSONValue) Interface() interface{} {
	return nil
}

func (self *JSONBool) Interface() interface{} {
	return self.data
}

func (self *JSONInt) Interface() interface{} {
	return self.data
}

func (self *JSONFloat) Interface() interface{} {
	return self.data
}

func (self *JSONString) Interface() interface{} {
	return self.data
}

func (self *JSONArray) Interface() interface{} {
	ret := make([]interface{}, len(self.data))
	for i := 0; i < len(self.data); i += 1 {
		ret[i] = self.data[i].Interface()
	}
	return ret
}

func (self *JSONDict) Interface() interface{} {
	mapping := make(map[string]interface{})

	for k, v := range self.data {
		mapping[k] = v.Interface()
	}

	return mapping
}
