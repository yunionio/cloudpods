// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import (
	"context"
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type Caller struct {
	modelVal reflect.Value
	funcName string
	inputs   []interface{}

	funcVal reflect.Value
}

func NewCaller(model interface{}, fName string) *Caller {
	return &Caller{
		modelVal: reflect.ValueOf(model),
		funcName: fName,
	}
}

func (c *Caller) Inputs(inputs ...interface{}) *Caller {
	c.inputs = inputs
	return c
}

func (c *Caller) Call() ([]reflect.Value, error) {
	return callObject(c.modelVal, c.funcName, c.inputs...)
}

func call(obj interface{}, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	return callObject(reflect.ValueOf(obj), fName, inputs...)
}

func callObject(modelVal reflect.Value, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	funcVal := modelVal.MethodByName(fName)
	return callFunc(funcVal, inputs...)
}

func callFunc(funcVal reflect.Value, inputs ...interface{}) ([]reflect.Value, error) {
	fName := funcVal.String()
	if !funcVal.IsValid() || funcVal.IsNil() {
		return nil, httperrors.NewActionNotFoundError(fmt.Sprintf("%s method not found", fName))
	}
	funcType := funcVal.Type()
	paramLen := funcType.NumIn()
	if paramLen != len(inputs) {
		return nil, httperrors.NewInternalServerError("%s method params length not match, expected %d, input %d", fName, paramLen, len(inputs))
	}
	params := make([]*param, paramLen)
	for i := range inputs {
		params[i] = newParam(funcType.In(i), inputs[i])
	}
	args := convertParams(params)
	return funcVal.Call(args), nil
}

func convertParams(params []*param) []reflect.Value {
	ret := make([]reflect.Value, 0)
	for _, p := range params {
		ret = append(ret, p.convert())
	}
	return ret
}

type param struct {
	pType reflect.Type
	input interface{}
}

func newParam(pType reflect.Type, input interface{}) *param {
	return &param{
		pType: pType,
		input: input,
	}
}

func isJSONObject(input interface{}) (jsonutils.JSONObject, bool) {
	val := reflect.ValueOf(input)
	obj, ok := val.Interface().(jsonutils.JSONObject)
	if !ok {
		return nil, false
	}
	return obj, true
}

func (p *param) convert() reflect.Value {
	if p.input == nil {
		return reflect.New(p.pType).Elem()
	}
	obj, ok := isJSONObject(p.input)
	if !ok {
		return reflect.ValueOf(p.input)
	}
	// generate object by type
	val := reflect.New(p.pType)
	obj.Unmarshal(val.Interface())
	return val.Elem()
}

func ValueToJSONObject(out reflect.Value) jsonutils.JSONObject {
	if obj, ok := isJSONObject(out); ok {
		return obj
	}
	return jsonutils.Marshal(out.Interface())
}

func ValueToError(out reflect.Value) error {
	errVal := out.Interface()
	if !gotypes.IsNil(errVal) {
		return errVal.(error)
	}
	return nil
}

func mergeInputOutputData(data *jsonutils.JSONDict, resVal reflect.Value) *jsonutils.JSONDict {
	retJson := ValueToJSONObject(resVal).(*jsonutils.JSONDict)
	// preserve the input info not returned by caller
	data.Update(retJson)
	return data
}

func ValidateCreateData(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret, err := call(manager, "ValidateCreateData", ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald ValidateCreateData return value")
	}
	resVal := ret[0]
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return mergeInputOutputData(data, resVal), nil
}

func ListItemFilter(manager IModelManager, ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	ret, err := call(manager, "ListItemFilter", ctx, q, userCred, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald ListItemFilter return value")
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(*sqlchemy.SQuery), nil
}

func GetExtraDetails(model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	ret, err := call(model, "GetExtraDetails", ctx, userCred, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald GetExtraDetails return value")
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ValueToJSONObject(ret[0]).(*jsonutils.JSONDict), nil
}

func ValidateUpdateData(model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret, err := call(model, "ValidateUpdateData", ctx, userCred, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald ValidateUpdateData return value")
	}
	resVal := ret[0]
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return mergeInputOutputData(data, resVal), nil
}

func CustomizeDelete(model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	ret, err := call(model, "CustomizeDelete", ctx, userCred, query, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return httperrors.NewInternalServerError("Invald CustomizeDelete return value")
	}
	return ValueToError(ret[0])
}
