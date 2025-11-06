package log

import (
	"fmt"
	"os"
	"time"
)

var DefaultTimeFormatter = func() string {
	return time.Now().Format(timeFmt)
}

// Preferred and probably faster than DefaultTimeFormatter.
var DefaultTimeAppendFormatter = func(b []byte) []byte {
	return time.Now().AppendFormat(b, timeFmt)
}

func GetDefaultTimeAppendFormatter() func([]byte) []byte {
	if DefaultTimeAppendFormatter != nil {
		return DefaultTimeAppendFormatter
	}
	// Hopefully this doesn't allocate.
	return func(b []byte) []byte {
		return append(b, DefaultTimeFormatter()...)
	}
}

var timeFmt string

func init() {
	var ok bool
	timeFmt, ok = os.LookupEnv(EnvTimeFormat)
	if !ok {
		timeFmt = "2006-01-02 15:04:05 -0700"
	}
}

var started = time.Now()

var TimeFormatSecondsSinceInit = func() string {
	return fmt.Sprintf("%.3fs", time.Since(started).Seconds())
}

var TimeAppendFormatSecondsSinceInit = func(b []byte) []byte {
	return fmt.Appendf(b, "%.3fs", time.Since(started).Seconds())
}
