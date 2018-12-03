package utils

import (
	"io"
	"runtime/pprof"
)

func DumpAllGoroutineStack(w io.Writer) {
	pprof.Lookup("goroutine").WriteTo(w, 1)
}
