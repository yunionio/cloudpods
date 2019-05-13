package tokens

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
)

func authMethodStr2Id(method string) byte {
	for i := range api.AUTH_METHODS {
		if api.AUTH_METHODS[i] == method {
			return byte(i) + 1
		}
	}
	return 0
}

func authMethodId2Str(mid byte) string {
	if mid >= 1 && int(mid) <= len(api.AUTH_METHODS) {
		return api.AUTH_METHODS[mid-1]
	}
	return ""
}

func authMethodsStr2Id(methods []string) []byte {
	ret := make([]byte, len(methods))
	for i := range methods {
		ret[i] = authMethodStr2Id(methods[i])
	}
	return ret
}

func authMethodsId2Str(mids []byte) []string {
	ret := make([]string, len(mids))
	for i := range mids {
		ret[i] = authMethodId2Str(mids[i])
	}
	return ret
}
