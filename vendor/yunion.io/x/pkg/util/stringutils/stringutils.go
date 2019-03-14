package stringutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-plus/uuid"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"
)

func ParseNamePattern(name string) (string, string, int) {
	const RepChar = '#'
	var match string
	var pattern string
	var patternLen int

	start := strings.IndexByte(name, RepChar)
	if start >= 0 {
		end := start + 1
		for end < len(name) && name[end] == RepChar {
			end += 1
		}
		match = fmt.Sprintf("%s%%%s", name[:start], name[end:])
		pattern = fmt.Sprintf("%s%%0%dd%s", name[:start], end-start, name[end:])
		patternLen = end - start
	} else {
		match = fmt.Sprintf("%s-%%", name)
		pattern = fmt.Sprintf("%s-%%d", name)
	}
	return match, pattern, patternLen
}

func UUID4() string {
	uid, _ := uuid.NewV4()
	return uid.String()
}

func Interface2String(val interface{}) string {
	if val == nil {
		return ""
	}
	switch val.(type) {
	case string:
		return val.(string)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%d", val)
	case jsonutils.JSONObject:
		json := val.(jsonutils.JSONObject)
		return json.String()
	case time.Time:
		tm := val.(time.Time)
		return timeutils.FullIsoTime(tm)
	default:
		if s, ok := val.(fmt.Stringer); ok {
			return s.String()
		}
		json := jsonutils.Marshal(val)
		return json.String()
	}
}

func SplitKeyValue(line string) (string, string) {
	return SplitKeyValueBySep(line, ":")
}

func SplitKeyValueBySep(line string, sep string) (string, string) {
	pos := strings.Index(line, sep)
	if pos > 0 {
		key := strings.TrimSpace(line[:pos])
		val := strings.TrimSpace(line[pos+1:])
		return key, val
	}
	return "", ""
}
