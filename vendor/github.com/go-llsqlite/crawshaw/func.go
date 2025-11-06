// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package sqlite

// #include <stdint.h>
// #include <stdlib.h>
// #include <sqlite3.h>
// #include "wrappers.h"
//
// static int go_sqlite3_create_function_v2(
//   sqlite3 *db,
//   const char *zFunctionName,
//   int nArg,
//   int eTextRep,
//   uintptr_t pApp,
//   void (*xFunc)(sqlite3_context*,int,sqlite3_value**),
//   void (*xStep)(sqlite3_context*,int,sqlite3_value**),
//   void (*xFinal)(sqlite3_context*),
//   void(*xDestroy)(void*)
// ) {
//   return sqlite3_create_function_v2(
//     db,
//     zFunctionName,
//     nArg,
//     eTextRep,
//     (void *)pApp,
//     xFunc,
//     xStep,
//     xFinal,
//     xDestroy);
// }
import "C"
import (
	"sync"
	"unsafe"
)

// Context is an *sqlite3_context.
// It is used by custom functions to return result values.
// An SQLite context is in no way related to a Go context.Context.
type Context struct {
	ptr *C.sqlite3_context
}

func (ctx Context) UserData() interface{} {
	return getxfuncs(ctx.ptr).data
}

func (ctx Context) SetUserData(data interface{}) {
	getxfuncs(ctx.ptr).data = data
}

func (ctx Context) ResultInt(v int)        { C.sqlite3_result_int(ctx.ptr, C.int(v)) }
func (ctx Context) ResultInt64(v int64)    { C.sqlite3_result_int64(ctx.ptr, C.sqlite3_int64(v)) }
func (ctx Context) ResultFloat(v float64)  { C.sqlite3_result_double(ctx.ptr, C.double(v)) }
func (ctx Context) ResultNull()            { C.sqlite3_result_null(ctx.ptr) }
func (ctx Context) ResultValue(v Value)    { C.sqlite3_result_value(ctx.ptr, v.ptr) }
func (ctx Context) ResultZeroBlob(n int64) { C.sqlite3_result_zeroblob64(ctx.ptr, C.sqlite3_uint64(n)) }
func (ctx Context) ResultBlob(v []byte) {
	C.sqlite3_result_blob(ctx.ptr, C.CBytes(v), C.int(len(v)), (*[0]byte)(C.cfree))
}
func (ctx Context) ResultText(v string) {
	var cv *C.char
	if len(v) != 0 {
		cv = C.CString(v)
	}
	C.sqlite3_result_text(ctx.ptr, cv, C.int(len(v)), (*[0]byte)(C.cfree))
}
func (ctx Context) ResultError(err error) {
	if err, isError := err.(Error); isError {
		C.sqlite3_result_error_code(ctx.ptr, C.int(err.Code))
		return
	}
	errstr := err.Error()
	cerrstr := C.CString(errstr)
	defer C.free(unsafe.Pointer(cerrstr))
	C.sqlite3_result_error(ctx.ptr, cerrstr, C.int(len(errstr)))
}

type Value struct {
	ptr *C.sqlite3_value
}

func (v Value) IsNil() bool      { return v.ptr == nil }
func (v Value) Int() int         { return int(C.sqlite3_value_int(v.ptr)) }
func (v Value) Int64() int64     { return int64(C.sqlite3_value_int64(v.ptr)) }
func (v Value) Float() float64   { return float64(C.sqlite3_value_double(v.ptr)) }
func (v Value) Len() int         { return int(C.sqlite3_value_bytes(v.ptr)) }
func (v Value) Type() ColumnType { return ColumnType(C.sqlite3_value_type(v.ptr)) }
func (v Value) Text() string {
	ptr := unsafe.Pointer(C.sqlite3_value_text(v.ptr))
	n := v.Len()
	return C.GoStringN((*C.char)(ptr), C.int(n))
}
func (v Value) Blob() []byte {
	ptr := unsafe.Pointer(C.sqlite3_value_blob(v.ptr))
	n := v.Len()
	return C.GoBytes(ptr, C.int(n))
}

type xfunc struct {
	id     int
	name   string
	conn   *Conn
	xFunc  func(Context, ...Value)
	xStep  func(Context, ...Value)
	xFinal func(Context)
	data   interface{}
}

var xfuncs = struct {
	mu   sync.RWMutex
	m    map[int]*xfunc
	next int
}{
	m: make(map[int]*xfunc),
}

// CreateFunction registers a Go function with SQLite
// for use in SQL queries.
//
// To define a scalar function, provide a value for
// xFunc and set xStep/xFinal to nil.
//
// To define an aggregation set xFunc to nil and
// provide values for xStep and xFinal.
//
// State can be stored across function calls by
// using the Context UserData/SetUserData methods.
//
// https://sqlite.org/c3ref/create_function.html
func (conn *Conn) CreateFunction(name string, deterministic bool, numArgs int, xFunc, xStep func(Context, ...Value), xFinal func(Context)) error {
	cname := C.CString(name) // TODO: free?
	eTextRep := C.int(C.SQLITE_UTF8)
	if deterministic {
		eTextRep |= C.SQLITE_DETERMINISTIC
	}

	x := &xfunc{
		conn:   conn,
		name:   name,
		xFunc:  xFunc,
		xStep:  xStep,
		xFinal: xFinal,
	}

	xfuncs.mu.Lock()
	xfuncs.next++
	x.id = xfuncs.next
	xfuncs.m[x.id] = x
	xfuncs.mu.Unlock()

	pApp := C.uintptr_t(x.id)

	var funcfn, stepfn, finalfn *[0]byte
	if xFunc == nil {
		stepfn = (*[0]byte)(C.c_step_tramp)
		finalfn = (*[0]byte)(C.c_final_tramp)
	} else {
		funcfn = (*[0]byte)(C.c_func_tramp)
	}

	res := C.go_sqlite3_create_function_v2(
		conn.conn,
		cname,
		C.int(numArgs),
		eTextRep,
		pApp,
		funcfn,
		stepfn,
		finalfn,
		(*[0]byte)(C.c_destroy_tramp),
	)
	return conn.reserr("Conn.CreateFunction", name, res)
}

func getxfuncs(ctx *C.sqlite3_context) *xfunc {
	id := int(uintptr(C.sqlite3_user_data(ctx)))

	xfuncs.mu.RLock()
	x := xfuncs.m[id]
	xfuncs.mu.RUnlock()

	return x
}

//export go_func_tramp
func go_func_tramp(ctx *C.sqlite3_context, n C.int, valarray **C.sqlite3_value) {
	x := getxfuncs(ctx)
	var vals []Value
	if n > 0 {
		vals = (*[127]Value)(unsafe.Pointer(valarray))[:n:n]
	}
	x.xFunc(Context{ptr: ctx}, vals...)
}

//export go_step_tramp
func go_step_tramp(ctx *C.sqlite3_context, n C.int, valarray **C.sqlite3_value) {
	x := getxfuncs(ctx)
	var vals []Value
	if n > 0 {
		vals = (*[127]Value)(unsafe.Pointer(valarray))[:n:n]
	}
	x.xStep(Context{ptr: ctx}, vals...)
}

//export go_final_tramp
func go_final_tramp(ctx *C.sqlite3_context) {
	x := getxfuncs(ctx)
	x.xFinal(Context{ptr: ctx})
}

//export go_destroy_tramp
func go_destroy_tramp(ptr uintptr) {
	id := int(ptr)

	xfuncs.mu.Lock()
	delete(xfuncs.m, id)
	xfuncs.mu.Unlock()
}
