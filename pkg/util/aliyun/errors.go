package aliyun

import (
	"fmt"
	"strings"
)

func isError(err error, code string) bool {
	errStr := fmt.Sprintf("%s", err)
	if strings.Index(errStr, code) > 0 {
		return true
	} else {
		return false
	}
}
