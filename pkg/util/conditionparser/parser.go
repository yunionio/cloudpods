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

package conditionparser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

var (
	ErrInvalidOp      = errors.Error("invalid operation")
	ErrFieldNotFound  = errors.Error("field not found")
	ErrOutOfIndex     = errors.Error("out of index")
	ErrFuncArgument   = errors.Error("invalid function arguments")
	ErrMethodNotFound = errors.Error("method not found")
)

func IsValid(exprStr string) bool {
	_, err := parser.ParseExpr(exprStr)
	if err != nil {
		return false
	}
	return true
}

func EvalString(exprStr string, input interface{}) (string, error) {
	if len(exprStr) == 0 {
		return "", nil
	}
	expr, err := parser.ParseExpr(exprStr)
	if err != nil {
		return "", errors.Wrapf(err, "parse expr %s error", exprStr)
	}
	result, err := eval(expr, input)
	if err != nil {
		return "", errors.Wrap(err, "eval")
	}
	switch strVal := result.(type) {
	case string:
		return strVal, nil
	case *jsonutils.JSONString:
		return strVal.GetString()
	default:
		return fmt.Sprintf("%s", strVal), nil
	}
}

func EvalBool(exprStr string, input interface{}) (bool, error) {
	if len(exprStr) == 0 {
		return true, nil
	}
	expr, err := parser.ParseExpr(exprStr)
	if err != nil {
		return false, errors.Wrapf(err, "parse expr %s", exprStr)
	}
	result, err := eval(expr, input)
	if err != nil {
		return false, errors.Wrap(err, "eval")
	}
	switch bVal := result.(type) {
	case bool:
		return bVal, nil
	case *jsonutils.JSONBool:
		return bVal.Bool()
	case []interface{}, *jsonutils.JSONArray:
		arrX := getArray(result)
		for i := 0; i < len(arrX); i += 1 {
			val := getBool(arrX[i])
			if val {
				return true, nil
			}
		}
	}
	return false, nil
}

func eval(expr ast.Expr, input interface{}) (interface{}, error) {
	switch expr.(type) {
	case *ast.BinaryExpr:
		return evalBinary(expr.(*ast.BinaryExpr), input)
	case *ast.UnaryExpr:
		return evalUnary(expr.(*ast.UnaryExpr), input)
	case *ast.SelectorExpr:
		return evalSelector(expr.(*ast.SelectorExpr), input)
	case *ast.Ident:
		return evalIdent(expr.(*ast.Ident), input)
	case *ast.ParenExpr:
		return evalParen(expr.(*ast.ParenExpr), input)
	case *ast.CallExpr:
		return evalCall(expr.(*ast.CallExpr), input)
	case *ast.IndexExpr:
		return evalIndex(expr.(*ast.IndexExpr), input)
	case *ast.BasicLit:
		return evalBasicLit(expr.(*ast.BasicLit), input)
	default:
		return nil, ErrInvalidOp
	}
}

func evalBasicLit(expr *ast.BasicLit, input interface{}) (interface{}, error) {
	switch expr.Kind {
	case token.IDENT:
		switch input.(type) {
		case *jsonutils.JSONDict:
			jsonX := input.(*jsonutils.JSONDict)
			return jsonX.Get(expr.Value)
		default:
			return nil, ErrInvalidOp
		}
	case token.INT:
		return strconv.Atoi(expr.Value)
	case token.FLOAT:
		return strconv.ParseFloat(expr.Value, 64)
	case token.CHAR:
		return expr.Value[1 : len(expr.Value)-1][0], nil
	case token.STRING:
		return expr.Value[1 : len(expr.Value)-1], nil
	default:
		return nil, ErrInvalidOp
	}
}

func evalIndex(expr *ast.IndexExpr, input interface{}) (interface{}, error) {
	X, err := eval(expr.X, input)
	if err != nil {
		return nil, err
	}
	indexV, err := eval(expr.Index, input)
	if err != nil {
		return nil, err
	}
	switch X.(type) {
	case []interface{}, *jsonutils.JSONArray:
		arrX := getArray(X)
		indexI := getInt(indexV)
		if indexI >= 0 && indexI < int64(len(arrX)) {
			return arrX[indexI], nil
		} else {
			return nil, ErrOutOfIndex
		}
	case *jsonutils.JSONDict:
		jsonX := X.(*jsonutils.JSONDict)
		field := getString(indexV)
		return getJSONProperty(jsonX, field)
	case string:
		strX := X.(string)
		indexI := getInt(indexV)
		if indexI >= 0 && indexI < int64(len(strX)) {
			return strX[indexI], nil
		} else {
			return nil, ErrOutOfIndex
		}
	default:
		return nil, err
	}
}

func args2Strings(args []interface{}) []string {
	strs := make([]string, len(args))
	for i := 0; i < len(args); i += 1 {
		strs[i] = getString(args[i])
	}
	return strs
}

func args2Ints(args []interface{}) []int {
	ints := make([]int, len(args))
	for i := 0; i < len(args); i += 1 {
		ints[i] = int(getInt(args[i]))
	}
	return ints
}

func evalCallInternal(funcV interface{}, args []interface{}) (interface{}, error) {
	switch funcV.(type) {
	case []interface{}, *jsonutils.JSONArray:
		arrFuncV := getArray(funcV)
		ret := make([]interface{}, len(arrFuncV))
		for i := 0; i < len(arrFuncV); i += 1 {
			reti, err := evalCallInternal(arrFuncV[i], args)
			if err != nil {
				return nil, err
			}
			ret[i] = reti
		}
		return ret, nil
	case *funcCaller:
		caller := funcV.(*funcCaller)
		switch caller.caller.(type) {
		case string:
			strX := caller.caller.(string)
			switch caller.method {
			case "startswith":
				strArgs := args2Strings(args)
				if len(strArgs) != 1 {
					return nil, ErrFuncArgument
				}
				return strings.HasPrefix(strX, strArgs[0]), nil
			case "endswith":
				strArgs := args2Strings(args)
				if len(strArgs) != 1 {
					return nil, ErrFuncArgument
				}
				return strings.HasSuffix(strX, strArgs[0]), nil
			case "contains":
				strArgs := args2Strings(args)
				if len(strArgs) != 1 {
					return nil, ErrFuncArgument
				}
				return strings.Contains(strX, strArgs[0]), nil
			case "in":
				var strArgs []string
				if len(args) == 0 {
					return nil, ErrFuncArgument
				} else if len(args) == 1 {
					switch args[0].(type) {
					case string, *jsonutils.JSONString:
						strArgs = []string{args[0].(string)}
					case []interface{}, *jsonutils.JSONArray:
						strArgs = args2Strings(getArray(args[0]))
					default:
						return nil, ErrFuncArgument
					}
				} else {
					strArgs = args2Strings(args)
				}
				return utils.IsInStringArray(strX, strArgs), nil
			case "len":
				if len(args) > 0 {
					return nil, ErrFuncArgument
				}
				return len(strX), nil
			case "substr":
				intArgs := args2Ints(args)
				var o1, o2 int
				if len(intArgs) == 1 {
					o1 = 0
					o2 = intArgs[0]
				} else if len(intArgs) == 2 {
					o1 = intArgs[0]
					o2 = intArgs[1]
				} else {
					return nil, ErrFuncArgument
				}
				if o1 < 0 {
					o1 = len(strX) + o1
				}
				if o2 < 0 {
					o2 = len(strX) + o2
				}
				if o1 < 0 || o1 >= len(strX) {
					return nil, ErrFuncArgument
				}
				if o2 <= o1 || o2 > len(strX) {
					return nil, ErrFuncArgument
				}
				return strX[o1:o2], nil
			default:
				return nil, ErrInvalidOp
			}
		case *jsonutils.JSONDict:
			jsonX := caller.caller.(*jsonutils.JSONDict)
			switch caller.method {
			case "contains":
				strArgs := args2Strings(args)
				return jsonX.Contains(strArgs...), nil
			case "len":
				if len(args) > 0 {
					return nil, ErrFuncArgument
				}
				return jsonX.Size(), nil
			case "keys":
				if len(args) > 0 {
					return nil, ErrFuncArgument
				}
				return jsonutils.NewStringArray(jsonX.SortedKeys()), nil
			default:
				return nil, ErrMethodNotFound
			}
		case []interface{}, *jsonutils.JSONArray:
			array := getArray(caller.caller)
			switch caller.method {
			case "len":
				if len(args) > 0 {
					return nil, ErrFuncArgument
				}
				return len(array), nil
			case "contains":
				if len(args) < 1 {
					return nil, ErrFuncArgument
				}
				for j := 0; j < len(args); j += 1 {
					find := false
					for i := 0; i < len(array); i += 1 {
						findInf, err := evalBinaryInternal(array[i], args[j], token.EQL)
						if err != nil {
							return nil, err
						}
						switch findInf.(type) {
						case bool:
							if findInf.(bool) {
								find = true
							}
						default:
							return nil, ErrInvalidOp
						}
					}
					if !find {
						return false, nil
					}
				}
				return true, nil
			default:
				return nil, ErrMethodNotFound
			}
		default:
			return nil, ErrInvalidOp
		}
	default:
		return nil, ErrInvalidOp
	}
}

func evalCall(expr *ast.CallExpr, input interface{}) (interface{}, error) {
	funcV, err := eval(expr.Fun, input)
	if err != nil {
		return nil, err
	}
	args := make([]interface{}, len(expr.Args))
	for i := 0; i < len(expr.Args); i += 1 {
		args[i], err = eval(expr.Args[i], input)
		if err != nil {
			return nil, err
		}
	}
	return evalCallInternal(funcV, args)
}

func evalParen(expr *ast.ParenExpr, input interface{}) (interface{}, error) {
	return eval(expr.X, input)
}

func getJSONProperty(json *jsonutils.JSONDict, identStr string) (jsonutils.JSONObject, error) {
	if json.Contains(identStr) {
		return json.Get(identStr)
	} else {
		identArray := jsonutils.GetArrayOfPrefix(json, identStr)
		if len(identArray) > 0 {
			return jsonutils.NewArray(identArray...), nil
		} else {
			return nil, ErrFieldNotFound
		}
	}
}

func evalIdent(expr *ast.Ident, input interface{}) (interface{}, error) {
	if expr.Obj == nil || input == nil {
		return expr.Name, nil
	} else {
		switch input.(type) {
		case *jsonutils.JSONDict:
			json := input.(*jsonutils.JSONDict)
			return getJSONProperty(json, expr.Name)
		default:
			return nil, ErrInvalidOp
		}
	}
}

type funcCaller struct {
	caller interface{}
	method string
}

func getArray(X interface{}) []interface{} {
	switch X.(type) {
	case *jsonutils.JSONArray:
		arr := X.(*jsonutils.JSONArray)
		ret := make([]interface{}, arr.Size())
		for i := 0; i < arr.Size(); i += 1 {
			ret[i], _ = arr.GetAt(i)
		}
		return ret
	default:
		return X.([]interface{})
	}
}

func evalSelectorInternal(X interface{}, identStr string) (interface{}, error) {
	switch X.(type) {
	case *jsonutils.JSONDict:
		json := X.(*jsonutils.JSONDict)
		ret, err := getJSONProperty(json, identStr)
		if err == ErrFieldNotFound {
			return &funcCaller{caller: X, method: identStr}, nil
		} else {
			return ret, err
		}
	case []interface{}, *jsonutils.JSONArray:
		if identStr == "len" || identStr == "contains" {
			return &funcCaller{caller: X, method: identStr}, nil
		}
		arrX := getArray(X)
		ret := make([]interface{}, len(arrX))
		for i := 0; i < len(arrX); i += 1 {
			reti, err := evalSelectorInternal(arrX[i], identStr)
			if err != nil {
				return nil, err
			}
			ret[i] = reti
		}
		return ret, nil
	case string, *jsonutils.JSONString:
		return &funcCaller{caller: getString(X), method: identStr}, nil
	default:
		return nil, ErrInvalidOp
	}
}

func evalSelector(expr *ast.SelectorExpr, input interface{}) (interface{}, error) {
	X, err := eval(expr.X, input)
	if err != nil {
		return nil, ErrInvalidOp
	}
	ident, err := evalIdent(expr.Sel, nil)
	if err != nil {
		return nil, ErrInvalidOp
	}
	identStr := ident.(string)

	return evalSelectorInternal(X, identStr)
}

func evalUnaryInternal(X interface{}, op token.Token) (interface{}, error) {
	switch X.(type) {
	case []interface{}, *jsonutils.JSONArray:
		arrX := getArray(X)
		ret := make([]interface{}, len(arrX))
		for i := 0; i < len(arrX); i += 1 {
			reti, err := evalUnaryInternal(arrX[i], op)
			if err != nil {
				return nil, err
			}
			ret[i] = reti
		}
		return ret, nil
	case bool, *jsonutils.JSONBool:
		boolX := getBool(X)
		switch op {
		case token.NOT:
			return !boolX, nil
		default:
			return nil, ErrInvalidOp
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *jsonutils.JSONInt:
		intX := getInt(X)
		switch op {
		case token.SUB:
			return -intX, nil
		default:
			return nil, ErrInvalidOp
		}
	case float32, float64, *jsonutils.JSONFloat:
		floatX := getFloat(X)
		switch op {
		case token.SUB:
			return -floatX, nil
		default:
			return nil, ErrInvalidOp
		}
	default:
		return nil, ErrInvalidOp
	}
}

func evalUnary(expr *ast.UnaryExpr, input interface{}) (interface{}, error) {
	X, err := eval(expr.X, input)
	if err != nil {
		return nil, err
	}
	op := expr.Op
	return evalUnaryInternal(X, op)
}

func getString(val interface{}) string {
	switch val.(type) {
	case *jsonutils.JSONString:
		jsonVal, _ := val.(*jsonutils.JSONString).GetString()
		return jsonVal
	default:
		return val.(string)
	}
}

func getBool(val interface{}) bool {
	switch val.(type) {
	case *jsonutils.JSONBool:
		jsonVal, _ := val.(*jsonutils.JSONBool).Bool()
		return jsonVal
	default:
		return val.(bool)
	}
}

func getInt(val interface{}) int64 {
	switch val.(type) {
	case *jsonutils.JSONInt:
		jsonVal, _ := val.(*jsonutils.JSONInt).Int()
		return jsonVal
	default:
		return reflect.ValueOf(val).Int()
	}
}

func getFloat(val interface{}) float64 {
	switch val.(type) {
	case *jsonutils.JSONFloat:
		jsonVal, _ := val.(*jsonutils.JSONFloat).Float()
		return jsonVal
	default:
		return reflect.ValueOf(val).Float()
	}
}

func evalBinaryInternal(X, Y interface{}, op token.Token) (interface{}, error) {
	switch X.(type) {
	case []interface{}, *jsonutils.JSONArray:
		arrX := getArray(X)
		ret := make([]interface{}, len(arrX))
		for i := 0; i < len(arrX); i += 1 {
			reti, err := evalBinaryInternal(arrX[i], Y, op)
			if err != nil {
				return nil, err
			}
			ret[i] = reti
		}
		return ret, nil
	}
	switch Y.(type) {
	case []interface{}, *jsonutils.JSONArray:
		arrY := getArray(Y)
		ret := make([]interface{}, len(arrY))
		for i := 0; i < len(arrY); i += 1 {
			reti, err := evalBinaryInternal(X, arrY[i], op)
			if err != nil {
				return nil, err
			}
			ret[i] = reti
		}
		return ret, nil
	}
	switch X.(type) {
	case bool, *jsonutils.JSONBool:
		switch Y.(type) {
		case bool, *jsonutils.JSONBool:
			boolX := getBool(X)
			boolY := getBool(Y)
			switch op {
			case token.LAND:
				return boolX && boolY, nil
			case token.LOR:
				return boolX || boolY, nil
			default:
				return nil, ErrInvalidOp
			}
		default:
			return nil, ErrInvalidOp
		}
	case string, *jsonutils.JSONString:
		switch Y.(type) {
		case string, *jsonutils.JSONString:
			strX := getString(X)
			strY := getString(Y)
			switch op {
			case token.ADD:
				return strX + strY, nil
			case token.EQL:
				return strX == strY, nil
			case token.NEQ:
				return strX != strY, nil
			default:
				return nil, ErrInvalidOp
			}
		default:
			return nil, ErrInvalidOp
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *jsonutils.JSONInt:
		intX := getInt(X)
		switch Y.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *jsonutils.JSONInt:
			intY := getInt(Y)
			return evalIntegerOp(intX, intY, op)
		case float32, float64, jsonutils.JSONFloat:
			floatX := float64(intX)
			floatY := getFloat(Y)
			return evalFloatOp(floatX, floatY, op)
		default:
			return nil, ErrInvalidOp
		}
	case float32, float64, *jsonutils.JSONFloat:
		floatX := getFloat(X)
		switch Y.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, *jsonutils.JSONInt:
			floatY := float64(getInt(Y))
			return evalFloatOp(floatX, floatY, op)
		case float32, float64, *jsonutils.JSONFloat:
			floatY := getFloat(Y)
			return evalFloatOp(floatX, floatY, op)
		default:
			return nil, ErrInvalidOp
		}
	default:
		return nil, ErrInvalidOp
	}
}

func evalBinary(bExpr *ast.BinaryExpr, input interface{}) (interface{}, error) {
	X, err := eval(bExpr.X, input)
	if err != nil {
		return nil, err
	}
	Y, err := eval(bExpr.Y, input)
	if err != nil {
		return nil, err
	}
	return evalBinaryInternal(X, Y, bExpr.Op)
}

func evalIntegerOp(X, Y int64, op token.Token) (interface{}, error) {
	switch op {
	case token.ADD:
		return X + Y, nil
	case token.SUB:
		return X - Y, nil
	case token.MUL:
		return X * Y, nil
	case token.QUO:
		return X / Y, nil
	case token.REM:
		return X % Y, nil
	case token.AND:
		return X & Y, nil
	case token.OR:
		return X | Y, nil
	case token.XOR:
		return X ^ Y, nil
	case token.SHL:
		return X << uint64(Y), nil
	case token.SHR:
		return X >> uint64(Y), nil
	case token.AND_NOT:
		return X &^ Y, nil
	case token.EQL:
		return X == Y, nil
	case token.LSS:
		return X < Y, nil
	case token.GTR:
		return X > Y, nil
	case token.NEQ:
		return X != Y, nil
	case token.LEQ:
		return X <= Y, nil
	case token.GEQ:
		return X >= Y, nil
	default:
		return nil, ErrInvalidOp
	}
}

func evalFloatOp(X, Y float64, op token.Token) (interface{}, error) {
	switch op {
	case token.ADD:
		return X + Y, nil
	case token.SUB:
		return X - Y, nil
	case token.MUL:
		return X * Y, nil
	case token.QUO:
		return X / Y, nil
	case token.EQL:
		return X == Y, nil
	case token.LSS:
		return X < Y, nil
	case token.GTR:
		return X > Y, nil
	case token.NEQ:
		return X != Y, nil
	case token.LEQ:
		return X <= Y, nil
	case token.GEQ:
		return X >= Y, nil
	default:
		return nil, ErrInvalidOp
	}
}
