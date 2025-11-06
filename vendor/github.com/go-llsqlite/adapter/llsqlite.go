package sqlite

type ColIter int

func (me *ColIter) PostInc() int {
	ret := int(*me)
	*me++
	return ret
}

func (me ColIter) Get() int {
	return int(me)
}

func IsResultCode(err error, code ResultCode) bool {
	actual, ok := GetResultCode(err)
	return ok && actual == code
}

func IsPrimaryResultCodeErr(err error, code ResultCode) bool {
	actual, ok := GetResultCode(err)
	return ok && actual.ToPrimary() == code.ToPrimary()
}
