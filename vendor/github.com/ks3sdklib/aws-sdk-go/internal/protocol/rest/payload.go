package rest

import (
	"reflect"
)

// PayloadMember returns the payload field member of i if there is one, or nil.
func PayloadMember(i interface{}) interface{} {
	if i == nil {
		return nil
	}

	v := reflect.ValueOf(i).Elem()
	if !v.IsValid() {
		return nil
	}
	if field, ok := v.Type().FieldByName("SDKShapeTraits"); ok {
		if payloadName := field.Tag.Get("payload"); payloadName != "" {
			field, _ := v.Type().FieldByName(payloadName)
			if field.Tag.Get("type") != "structure" {
				return nil
			}

			payload := v.FieldByName(payloadName)
			if payload.IsValid() || (payload.Kind() == reflect.Ptr && !payload.IsNil()) {
				return payload.Interface()
			}
		}
	}
	return nil
}

// PayloadType returns the type of a payload field member of i if there is one, or "".
func PayloadType(i interface{}) string {
	v := reflect.Indirect(reflect.ValueOf(i))
	if !v.IsValid() {
		return ""
	}
	if field, ok := v.Type().FieldByName("SDKShapeTraits"); ok {
		if payloadName := field.Tag.Get("payload"); payloadName != "" {
			if payloadName == "GetBucketCORSInput" {
				return field.Tag.Get("type")
			}
			if member, ok := v.Type().FieldByName(payloadName); ok {
				return member.Tag.Get("type")
			}
		}
	}
	return ""
}

// PayloadMd5 判断给定结构体 i 中是否有 AutoFillMD5 字段
func PayloadMd5(i interface{}) (hasField bool) {
	// 获取结构体指针的 Value
	v := reflect.Indirect(reflect.ValueOf(i))
	// 如果结构体不存在或为空，则直接返回 false
	if !v.IsValid() {
		return
	}
	// 判断是否存在 AutoFillMD5 字段
	_, hasField = v.Type().FieldByName("AutoFillMD5")
	return
}
