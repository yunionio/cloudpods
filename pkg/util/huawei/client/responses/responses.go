package responses

import (
	"regexp"
	"strings"
	"yunion.io/x/jsonutils"
)

type ListResult struct {
	Data   []jsonutils.JSONObject
	Total  int
	Limit  int
	Offset int
}

func ListResult2JSONWithKey(result *ListResult, key string) jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	if result.Total > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Total)), "total")
	}
	if result.Limit > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Limit)), "limit")
	}
	if result.Offset > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Offset)), "offset")
	}
	arr := jsonutils.NewArray(result.Data...)
	obj.Add(arr, key)
	return obj
}

func ListResult2JSON(result *ListResult) jsonutils.JSONObject {
	return ListResult2JSONWithKey(result, "data")
}

func JSON2ListResult(result jsonutils.JSONObject) *ListResult {
	total, _ := result.Int("total")
	limit, _ := result.Int("limit")
	offset, _ := result.Int("offset")
	data, _ := result.GetArray("data")
	return &ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}
}

// 将key中的冒号替换成
func TransColonToDot(obj jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	re, _ := regexp.Compile("[a-zA-Z0-9](:+)[^\"]+\"\\s*:\\s*")

	if obj == nil {
		return obj, nil
	}

	newStr := re.ReplaceAllStringFunc(obj.String(), func(s string) string {
		count := strings.Count(s, ":")
		if count > 1 {
			return strings.Replace(s, ":", ".", count-1)
		}
		return s
	})

	return jsonutils.ParseString(newStr)
}
