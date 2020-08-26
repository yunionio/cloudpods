package base64

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
)

func EncodeToStringStd(src []byte) string {
	return base64.RawStdEncoding.EncodeToString(src)
}

func EncodeToString(src []byte) string {
	return base64.RawURLEncoding.EncodeToString(src)
}

func EncodeUint64ToString(v uint64) string {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, v)

	i := 0
	for ; i < len(data); i++ {
		if data[i] != 0x0 {
			break
		}
	}

	return EncodeToString(data[i:])
}

func DecodeString(src string) ([]byte, error) {
	var isRaw = !strings.HasSuffix(src, "=")
	if strings.ContainsAny(src, "+/") {
		if isRaw {
			return base64.RawStdEncoding.DecodeString(src)
		}
		return base64.StdEncoding.DecodeString(src)
	}

	if isRaw {
		return base64.RawURLEncoding.DecodeString(src)
	}
	return base64.URLEncoding.DecodeString(src)
}
