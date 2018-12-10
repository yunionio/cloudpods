package qcloud

import (
	"fmt"
	"strings"
)

func isError(err error, codes []string) bool {
	errStr := fmt.Sprintf("%s", err)
	for _, code := range codes {
		if strings.Index(errStr, code) > 0 {
			return true
		}
	}
	return false
}
