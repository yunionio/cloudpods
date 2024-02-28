/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package parser

import "strconv"

const (
	MAX_DEC_LEN = 38
)

const (
	NORMAL int = iota
	INT
	DOUBLE
	DECIMAL
	STRING
	HEX_INT
	WHITESPACE_OR_COMMENT
	NULL
)

type LVal struct {
	Value    string
	Tp       int
	Position int
}

func newLValNoParams() *LVal {
	return new(LVal).reset()
}

func newLVal(value string, tp int) *LVal {
	return &LVal{Value: value, Tp: tp}
}

func (l *LVal) reset() *LVal {
	l.Value = ""
	l.Tp = NORMAL
	return l
}

func (l *LVal) String() string {
	return strconv.Itoa(l.Tp) + ":" + l.Value
}
