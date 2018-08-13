package tristate

import "reflect"

type TriState string

const (
	True  = TriState("true")
	False = TriState("false")
	None  = TriState("")
)

var (
	TriStateType       = reflect.TypeOf(True)
	TriStateTrueValue  = reflect.ValueOf(True)
	TriStateFalseValue = reflect.ValueOf(False)
	TriStateNoneValue  = reflect.ValueOf(None)
)

func (r TriState) Bool() bool {
	switch r {
	case True:
		return true
	case False:
		return false
	default:
		return false
	}
}

func (r TriState) String() string {
	return string(r)
}

func (r TriState) IsNone() bool {
	return r == None
}

func (r TriState) IsTrue() bool {
	return r == True
}

func (r TriState) IsFalse() bool {
	return r == False
}
