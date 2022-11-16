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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

func findFunc(modelVal reflect.Value, fName string) (reflect.Value, error) {
	funcVal := modelVal.MethodByName(fName)
	if !funcVal.IsValid() || funcVal.IsNil() {
		log.Debugf("find method %s for %s", fName, modelVal.Type())
		if modelVal.Kind() != reflect.Ptr {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s not implemented", fName)
		}
		modelVal = modelVal.Elem()
		if modelVal.Kind() != reflect.Struct {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s not implemented", fName)
		}
		modelType := modelVal.Type()
		for i := 0; i < modelType.NumField(); i += 1 {
			fieldType := modelType.Field(i)
			if fieldType.Anonymous {
				fieldValue := modelVal.Field(i)
				if fieldValue.Kind() != reflect.Ptr && fieldValue.CanAddr() {
					newFuncVal, err := findFunc(fieldValue.Addr(), fName)
					if err == nil {
						if !funcVal.IsValid() || funcVal.IsNil() {
							funcVal = newFuncVal
						} else {
							return funcVal, errors.Wrapf(httperrors.ErrConflict, "%s is ambiguous", fName)
						}
					}
				} else if fieldValue.Kind() == reflect.Ptr {
					newFuncVal, err := findFunc(fieldValue, fName)
					if err == nil {
						if !funcVal.IsValid() || funcVal.IsNil() {
							funcVal = newFuncVal
						} else {
							return funcVal, errors.Wrapf(httperrors.ErrConflict, "%s is ambiguous", fName)
						}
					}
				}
			}
		}
		if !funcVal.IsValid() || funcVal.IsNil() {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s is not implemented", fName)
		}
	}
	return funcVal, nil
}

func callObject(modelVal reflect.Value, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	funcVal := modelVal.MethodByName(fName)
	return callFunc(funcVal, fName, inputs...)
}

func callFunc(funcVal reflect.Value, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	if !funcVal.IsValid() || funcVal.IsNil() {
		return nil, httperrors.NewActionNotFoundError("%s method not found, please check service version, current version: %s", fName, version.GetShortString())
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
	args, err := convertParams(params)
	if err != nil {
		return nil, err
	}
	return funcVal.Call(args), nil
}

func convertParams(params []*param) ([]reflect.Value, error) {
	ret := make([]reflect.Value, 0)
	for _, p := range params {
		val, err := p.convert()
		if err != nil {
			return ret, err
		}
		ret = append(ret, val)
	}
	return ret, nil
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

func (p *param) convert() (reflect.Value, error) {
	if p.input == nil {
		return reflect.New(p.pType).Elem(), nil
	}
	obj, ok := isJSONObject(p.input)
	if !ok {
		return reflect.ValueOf(p.input), nil
	}
	// generate object by type
	val := reflect.New(p.pType)
	err := obj.Unmarshal(val.Interface())
	if err != nil {
		return reflect.Value{}, errors.Wrapf(err, "unable to convert '%v' to Type %q", p.input, p.pType.Name())
	}
	return val.Elem(), nil
}

func ValueToJSONObject(out reflect.Value) jsonutils.JSONObject {
	return _valueToJSONObject(out, false)
}

func _valueToJSONObject(out reflect.Value, allFields bool) jsonutils.JSONObject {
	if gotypes.IsNil(out.Interface()) {
		return nil
	}

	if obj, ok := isJSONObject(out); ok {
		return obj
	}
	if allFields {
		return jsonutils.MarshalAll(out.Interface())
	} else {
		return jsonutils.Marshal(out.Interface())
	}
}

func ValueToJSONDict(out reflect.Value) *jsonutils.JSONDict {
	return _valueToJSONDict(out, false)
}

func _valueToJSONDict(out reflect.Value, allFields bool) *jsonutils.JSONDict {
	jsonObj := _valueToJSONObject(out, allFields)
	if jsonObj == nil {
		return nil
	}
	return jsonObj.(*jsonutils.JSONDict)
}

func ValueToError(out reflect.Value) error {
	errVal := out.Interface()
	if !gotypes.IsNil(errVal) {
		return errVal.(error)
	}
	return nil
}

func mergeInputOutputData(input *jsonutils.JSONDict, resVal reflect.Value) *jsonutils.JSONDict {
	output := _valueToJSONDict(resVal, true)
	// preserve the input info not returned by caller
	ret := input.Copy()
	jsonMap, _ := output.GetMap()
	for k, v := range jsonMap {
		if input.Contains(k) && v == jsonutils.JSONNull {
			ret.Remove(k)
			continue
		}
		if v != jsonutils.JSONNull && !v.IsZero() {
			ret.Set(k, v)
		}
	}
	return ret
}

func ValidateCreateData(funcName string, manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret, err := call(manager, funcName, ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
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
		return nil, httperrors.NewInternalServerError("Invald ListItemFilter return value count %d", len(ret))
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(*sqlchemy.SQuery), nil
}

func OrderByExtraFields(
	manager IModelManager,
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (*sqlchemy.SQuery, error) {
	ret, err := call(manager, "OrderByExtraFields", ctx, q, userCred, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald OrderByExtraFields return value count %d", len(ret))
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(*sqlchemy.SQuery), nil
}

func FetchCustomizeColumns(
	manager IModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) ([]*jsonutils.JSONDict, error) {
	ret, err := call(manager, "FetchCustomizeColumns", ctx, userCred, query, objs, fields, isList)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value count %d", len(ret))
	}
	if ret[0].IsNil() {
		return nil, nil
	}
	if ret[0].Kind() != reflect.Slice {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value type, not a slice!")
	}
	if ret[0].Len() != len(objs) {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value, inconsistent obj count: input %d != output %d", len(objs), ret[0].Len())
	}

	showReason := false
	if query.Contains("show_fail_reason") {
		showReason = true
	}

	retVal := make([]*jsonutils.JSONDict, ret[0].Len())
	for i := 0; i < ret[0].Len(); i += 1 {
		jsonDict := ValueToJSONDict(ret[0].Index(i))
		// NOTE: don't use obj update jsonDict as retval
		jsonDict.Update(jsonutils.Marshal(objs[i]).(*jsonutils.JSONDict))
		out := apis.ModelBaseDetails{
			CanDelete: true,
			CanUpdate: true,
		}

		err = ValidateDeleteCondition(objs[i].(IModel), ctx, jsonDict)
		if err != nil {
			out.CanDelete = false
			if showReason {
				out.DeleteFailReason = httperrors.NewErrorFromGeneralError(ctx, err)
			}
		}
		err = ValidateUpdateCondition(objs[i].(IModel), ctx)
		if err != nil {
			out.CanUpdate = false
			if showReason {
				out.UpdateFailReason = httperrors.NewErrorFromGeneralError(ctx, err)
			}
		}
		jsonDict.Update(jsonutils.Marshal(out))

		retVal[i] = jsonDict

	}
	return retVal, nil
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

func ValidateDeleteCondition(model IModel, ctx context.Context, data jsonutils.JSONObject) error {
	ret, err := call(model, "ValidateDeleteCondition", ctx, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return httperrors.NewInternalServerError("Invald ValidateDeleteCondition return value")
	}
	return ValueToError(ret[0])
}

func ValidateUpdateCondition(model IModel, ctx context.Context) error {
	ret, err := call(model, "ValidateUpdateCondition", ctx)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return httperrors.NewInternalServerError("Invald ValidateUpdateCondition return value")
	}
	return ValueToError(ret[0])
}
