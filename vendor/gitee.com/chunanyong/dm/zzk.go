/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"strconv"
	"strings"

	"gitee.com/chunanyong/dm/parser"

	"gitee.com/chunanyong/dm/util"
)

func (dc *DmConnection) lex(sql string) ([]*parser.LVal, error) {
	if dc.lexer == nil {
		dc.lexer = parser.NewLexer(strings.NewReader(sql), false)
	} else {
		dc.lexer.Reset(strings.NewReader(sql))
	}

	lexer := dc.lexer
	var lval *parser.LVal
	var err error
	lvalList := make([]*parser.LVal, 0, 64)
	lval, err = lexer.Yylex()
	if err != nil {
		return nil, err
	}

	for lval != nil {
		lvalList = append(lvalList, lval)
		lval.Position = len(lvalList)
		lval, err = lexer.Yylex()
		if err != nil {
			return nil, err
		}
	}

	return lvalList, nil
}

func lexSkipWhitespace(sql string, n int) ([]*parser.LVal, error) {
	lexer := parser.NewLexer(strings.NewReader(sql), false)

	var lval *parser.LVal
	var err error
	lvalList := make([]*parser.LVal, 0, 64)
	lval, err = lexer.Yylex()
	if err != nil {
		return nil, err
	}

	for lval != nil && n > 0 {
		lval.Position = len(lvalList)
		if lval.Tp == parser.WHITESPACE_OR_COMMENT {
			continue
		}

		lvalList = append(lvalList, lval)
		n--
		lval, err = lexer.Yylex()
		if err != nil {
			return nil, err
		}

	}

	return lvalList, nil
}

func (dc *DmConnection) escape(sql string, keywords []string) (string, error) {

	if (keywords == nil || len(keywords) == 0) && strings.Index(sql, "{") == -1 {
		return sql, nil
	}
	var keywordMap map[string]interface{}
	if keywords != nil && len(keywords) > 0 {
		keywordMap = make(map[string]interface{}, len(keywords))
		for _, keyword := range keywords {
			keywordMap[strings.ToUpper(keyword)] = nil
		}
	}
	nsql := bytes.NewBufferString("")
	stack := make([]bool, 0, 64)
	lvalList, err := dc.lex(sql)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(lvalList); i++ {
		lval0 := lvalList[i]
		if lval0.Tp == parser.NORMAL {
			if lval0.Value == "{" {
				lval1 := next(lvalList, i+1)
				if lval1 == nil || lval1.Tp != parser.NORMAL {
					stack = append(stack, false)
					nsql.WriteString(lval0.Value)
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "escape") || util.StringUtil.EqualsIgnoreCase(lval1.Value, "call") {
					stack = append(stack, true)
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "oj") {
					stack = append(stack, true)
					lval1.Value = ""
					lval1.Tp = parser.WHITESPACE_OR_COMMENT
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "d") {
					stack = append(stack, true)
					lval1.Value = "date"
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "t") {
					stack = append(stack, true)
					lval1.Value = "time"
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "ts") {
					stack = append(stack, true)
					lval1.Value = "datetime"
				} else if util.StringUtil.EqualsIgnoreCase(lval1.Value, "fn") {
					stack = append(stack, true)
					lval1.Value = ""
					lval1.Tp = parser.WHITESPACE_OR_COMMENT
					lval2 := next(lvalList, lval1.Position+1)
					if lval2 != nil && lval2.Tp == parser.NORMAL && util.StringUtil.EqualsIgnoreCase(lval2.Value, "database") {
						lval2.Value = "cur_database"
					}
				} else if util.StringUtil.Equals(lval1.Value, "?") {
					lval2 := next(lvalList, lval1.Position+1)
					if lval2 != nil && lval2.Tp == parser.NORMAL && util.StringUtil.EqualsIgnoreCase(lval2.Value, "=") {
						lval3 := next(lvalList, lval2.Position+1)
						if lval3 != nil && lval3.Tp == parser.NORMAL && util.StringUtil.EqualsIgnoreCase(lval3.Value, "call") {
							stack = append(stack, true)
							lval3.Value = ""
							lval3.Tp = parser.WHITESPACE_OR_COMMENT
						} else {
							stack = append(stack, false)
							nsql.WriteString(lval0.Value)
						}
					} else {
						stack = append(stack, false)
						nsql.WriteString(lval0.Value)
					}
				} else {
					stack = append(stack, false)
					nsql.WriteString(lval0.Value)
				}
			} else if util.StringUtil.Equals(lval0.Value, "}") {
				if len(stack) != 0 && stack[len(stack)-1] {

				} else {
					nsql.WriteString(lval0.Value)
				}
				stack = stack[:len(stack)-1]
			} else {
				if keywordMap != nil {
					_, ok := keywordMap[strings.ToUpper(lval0.Value)]
					if ok {
						nsql.WriteString("\"" + util.StringUtil.ProcessDoubleQuoteOfName(strings.ToUpper(lval0.Value)) + "\"")
					} else {
						nsql.WriteString(lval0.Value)
					}
				} else {
					nsql.WriteString(lval0.Value)
				}
			}
		} else if lval0.Tp == parser.STRING {
			nsql.WriteString("'" + util.StringUtil.ProcessSingleQuoteOfName(lval0.Value) + "'")
		} else {
			nsql.WriteString(lval0.Value)
		}
	}

	return nsql.String(), nil
}

func next(lvalList []*parser.LVal, start int) *parser.LVal {
	var lval *parser.LVal

	size := len(lvalList)
	for i := start; i < size; i++ {
		lval = lvalList[i]
		if lval.Tp != parser.WHITESPACE_OR_COMMENT {
			break
		}
	}
	return lval
}

func (dc *DmConnection) execOpt(sql string, optParamList []OptParameter, serverEncoding string) (string, []OptParameter, error) {
	nsql := bytes.NewBufferString("")

	lvalList, err := dc.lex(sql)
	if err != nil {
		return "", optParamList, err
	}

	if nil == lvalList || len(lvalList) == 0 {
		return sql, optParamList, nil
	}

	firstWord := lvalList[0].Value
	if !(util.StringUtil.EqualsIgnoreCase(firstWord, "INSERT") || util.StringUtil.EqualsIgnoreCase(firstWord, "SELECT") ||
		util.StringUtil.EqualsIgnoreCase(firstWord, "UPDATE") || util.StringUtil.EqualsIgnoreCase(firstWord, "DELETE")) {
		return sql, optParamList, nil
	}

	breakIndex := 0
	for i := 0; i < len(lvalList); i++ {
		lval := lvalList[i]
		switch lval.Tp {
		case parser.NULL:
			{
				nsql.WriteString("?")
				optParamList = append(optParamList, newOptParameter(nil, NULL, NULL_PREC))
			}
		case parser.INT:
			{
				nsql.WriteString("?")
				value, err := strconv.Atoi(lval.Value)
				if err != nil {
					return "", optParamList, err
				}

				if value <= int(INT32_MAX) && value >= int(INT32_MIN) {
					optParamList = append(optParamList, newOptParameter(G2DB.toInt32(int32(value)), INT, INT_PREC))

				} else {
					optParamList = append(optParamList, newOptParameter(G2DB.toInt64(int64(value)), BIGINT, BIGINT_PREC))
				}
			}
		case parser.DOUBLE:
			{
				nsql.WriteString("?")
				f, err := strconv.ParseFloat(lval.Value, 64)
				if err != nil {
					return "", optParamList, err
				}

				optParamList = append(optParamList, newOptParameter(G2DB.toFloat64(f), DOUBLE, DOUBLE_PREC))
			}
		case parser.DECIMAL:
			{
				nsql.WriteString("?")
				bytes, err := G2DB.toDecimal(lval.Value, 0, 0)
				if err != nil {
					return "", optParamList, err
				}
				optParamList = append(optParamList, newOptParameter(bytes, DECIMAL, 0))
			}
		case parser.STRING:
			{

				if len(lval.Value) > int(INT16_MAX) {

					nsql.WriteString("'" + util.StringUtil.ProcessSingleQuoteOfName(lval.Value) + "'")
				} else {
					nsql.WriteString("?")
					optParamList = append(optParamList, newOptParameter(Dm_build_650.Dm_build_866(lval.Value, serverEncoding, dc), VARCHAR, VARCHAR_PREC))
				}
			}
		case parser.HEX_INT:

			nsql.WriteString(lval.Value)
		default:

			nsql.WriteString(lval.Value)
		}

		if breakIndex > 0 {
			break
		}
	}

	if breakIndex > 0 {
		for i := breakIndex + 1; i < len(lvalList); i++ {
			nsql.WriteString(lvalList[i].Value)
		}
	}

	return nsql.String(), optParamList, nil
}

func (dc *DmConnection) hasConst(sql string) (bool, error) {
	lvalList, err := dc.lex(sql)
	if err != nil {
		return false, err
	}

	if nil == lvalList || len(lvalList) == 0 {
		return false, nil
	}

	for i := 0; i < len(lvalList); i++ {
		switch lvalList[i].Tp {
		case parser.NULL, parser.INT, parser.DOUBLE, parser.DECIMAL, parser.STRING, parser.HEX_INT:
			return true, nil
		}
	}
	return false, nil
}

type OptParameter struct {
	bytes  []byte
	ioType byte
	tp     int
	prec   int
	scale  int
}

func newOptParameter(bytes []byte, tp int, prec int) OptParameter {
	o := new(OptParameter)
	o.bytes = bytes
	o.tp = tp
	o.prec = prec
	return *o
}

func (parameter *OptParameter) String() string {
	if parameter.bytes == nil {
		return ""
	}
	return string(parameter.bytes)
}
