package httputils

import (
	"bytes"
	"fmt"
)

// New a http Json client error
// code: http error code, >=400
// class: error class
// msg: message
// params: message format parameters
func NewJsonClientError(code int, class string, msg string, params ...interface{}) *JSONClientError {
	details, err := errorMessage(msg, params...)
	return &JSONClientError{Code: code, Class: class, Details: details, Data: err}
}

func errorMessage(msgFmt string, params ...interface{}) (string, Error) {
	fields := make([]string, len(params))
	for i, v := range params {
		fields[i] = fmt.Sprint(v)
	}

	err := Error{
		Id:     msgFmtToTmpl(msgFmt),
		Fields: fields,
	}

	msg := msgFmt
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return msg, err
}

func msgFmtToTmpl(msgFmt string) string {
	// 将%s %d之类格式化字符串转换成{0}、{1}格式
	// 注意： 1.不支持复杂类型的转换例如%.2f , %[1]d, % x
	//       2.原始msgFmt中如果包含{0},{1}形式的字符串同样会引发错误。
	// 在抛出error msgFmt时应注意避免
	fmtstr := false
	lst := []rune(msgFmt)
	lastIndex := len(lst) - 1
	temp := bytes.Buffer{}
	index := 0
	for i, c := range lst {
		switch c {
		case '%':
			if fmtstr || i == lastIndex {
				temp.WriteRune(c)
				fmtstr = false
			} else {
				fmtstr = true
			}
		case 'v', 'T', 't', 'b', 'c', 'd', 'o', 'q', 'x', 'X', 'U', 'e', 'E', 'f', 'F', 'g', 'G', 's', 'p':
			if fmtstr {
				temp.WriteRune('{')
				temp.WriteString(fmt.Sprintf("%d", index))
				temp.WriteRune('}')
				index++
				fmtstr = false
			} else {
				temp.WriteRune(c)
			}

		default:
			if fmtstr {
				temp.WriteRune('%')
			}
			temp.WriteRune(c)
			fmtstr = false
		}
	}

	return temp.String()
}

func MsgTmplToFmt(tmpl string) string {
	return msgTmplToFmt(tmpl)
}

func msgTmplToFmt(tmpl string) string {
	b := &bytes.Buffer{}
	for i := 0; i < len(tmpl); {
		r := tmpl[i]
		if r != '{' {
			b.WriteByte(r)
			i++
			continue
		}

		j := i + 1
		for ; j < len(tmpl); j++ {
			r := tmpl[j]
			if r < '0' || r > '9' {
				break
			}
		}
		if j == len(tmpl) {
			b.WriteString(tmpl[i:])
			return b.String()
		}
		if j > i+1 && tmpl[j] == '}' {
			b.WriteString("%s")
			i = j + 1
		} else {
			b.WriteString(tmpl[i:j])
			i = j
		}
	}
	return b.String()
}
