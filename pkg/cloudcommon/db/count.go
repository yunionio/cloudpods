package db

import (
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

func FetchModelExtraCountProperties(model IModel, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	selfValue := reflect.ValueOf(model)
	selfType := reflect.TypeOf(model)
	for i := 0; i < selfValue.NumMethod(); i += 1 {
		methodValue := selfValue.Method(i)
		methodType := methodValue.Type()
		if methodType.NumIn() != 0 || methodType.NumOut() != 2 {
			continue
		}
		methodName := selfType.Method(i).Name
		tokens := utils.CamelSplitTokens(methodName)
		if len(tokens) < 3 {
			continue
		}
		if strings.EqualFold(tokens[0], "get") && strings.EqualFold(tokens[len(tokens)-1], "count") {
			resName := strings.ToLower(strings.Join(tokens[1:], "_"))
			outs := methodValue.Call([]reflect.Value{})
			extra.Add(jsonutils.NewInt(outs[0].Int()), resName)
		}
	}
	return extra
}
