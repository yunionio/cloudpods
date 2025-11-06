package log

import (
	"fmt"
)

// A wrapper around MsgImpl that provides some extra helpers to modify a Msg.
type Msg struct {
	MsgImpl
}

func (m Msg) String() string {
	return m.Text()
}

func newMsg(text func() string) Msg {
	return Msg{rootMsgImpl{text}}
}

func Fmsg(format string, a ...interface{}) Msg {
	return newMsg(func() string { return fmt.Sprintf(format, a...) })
}

func Msgln(a ...interface{}) Msg {
	return newMsg(func() string {
		s := fmt.Sprintln(a...)
		// Sprintln adds a final newline, but we add that ourselves deeper in the logging code.
		return s[:len(s)-1]
	})
}

var Fstr = Fmsg

func Str(s string) (m Msg) {
	return newMsg(func() string { return s })
}

type msgSkipCaller struct {
	MsgImpl
	skip int
}

func (me msgSkipCaller) Callers(skip int, pc []uintptr) int {
	return me.MsgImpl.Callers(skip+1+me.skip, pc)
}

func (m Msg) Skip(skip int) Msg {
	return Msg{msgSkipCaller{m.MsgImpl, skip}}
}

type item struct {
	key, value interface{}
}

// rename sink
func (m Msg) Log(l Logger) Msg {
	l.Log(m.Skip(1))
	return m
}

func (m Msg) LogLevel(level Level, l Logger) Msg {
	l.LogLevel(level, m.Skip(1))
	return m
}

type msgWithValues struct {
	MsgImpl
	values []interface{}
}

func (me msgWithValues) Values(cb valueIterCallback) {
	for _, v := range me.values {
		if !cb(v) {
			return
		}
	}
	me.MsgImpl.Values(cb)
}

// TODO: What ordering should be applied to the values here, per MsgImpl.Values. For now they're
// traversed in order of the slice.
func (m Msg) WithValues(v ...interface{}) Msg {
	return Msg{msgWithValues{m.MsgImpl, v}}
}

func (m Msg) AddValues(v ...interface{}) Msg {
	return m.WithValues(v...)
}

func (m Msg) With(key, value interface{}) Msg {
	return m.WithValues(item{key, value})
}

func (m Msg) Add(key, value interface{}) Msg {
	return m.With(key, value)
}

//func (m Msg) SetLevel(level Level) Msg {
//	return m.With(levelKey, level)
//}

//func (m Msg) GetByKey(key interface{}) (value interface{}, ok bool) {
//	m.Values(func(i interface{}) bool {
//		if keyValue, isKeyValue := i.(item); isKeyValue && keyValue.key == key {
//			value = keyValue.value
//			ok = true
//		}
//		return !ok
//	})
//	return
//}

//func (m Msg) GetLevel() (l Level, ok bool) {
//	v, ok := m.GetByKey(levelKey)
//	if ok {
//		l = v.(Level)
//	}
//	return
//}

func (m Msg) HasValue(v interface{}) (has bool) {
	m.Values(func(i interface{}) bool {
		if i == v {
			has = true
		}
		return !has
	})
	return
}

func (m Msg) AddValue(v interface{}) Msg {
	return m.AddValues(v)
}

//func (m Msg) GetValueByType(p interface{}) bool {
//	pve := reflect.ValueOf(p).Elem()
//	t := pve.Type()
//	return !iter.All(func(i interface{}) bool {
//		iv := reflect.ValueOf(i)
//		if iv.Type() == t {
//			pve.Set(iv)
//			return false
//		}
//		return true
//	}, m.Values)
//}

func (m Msg) WithText(f func(Msg) string) Msg {
	return Msg{msgWithText{
		m,
		func() string { return f(m) },
	}}
}

type msgWithText struct {
	MsgImpl
	text func() string
}

func (me msgWithText) Text() string {
	return me.text()
}
