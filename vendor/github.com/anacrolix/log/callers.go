package log

import (
	"runtime"
	"strings"
	"sync"
)

func getSingleCallerPc(skip int) uintptr {
	var pc [1]uintptr
	runtime.Callers(skip+2, pc[:])
	return pc[0]
}

type Loc struct {
	Package  string
	Function string
	File     string
	Line     int
}

// This is the package returned for a caller frame that is in the main package for a binary.
const mainPackageFrameImport = "main"

func locFromPc(pc uintptr) Loc {
	f, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	lastSlash := strings.LastIndexByte(f.Function, '/')
	firstDot := strings.IndexByte(f.Function[lastSlash+1:], '.')
	pkg := f.Function[:lastSlash+1+firstDot]
	if pkg == mainPackageFrameImport {
		pkg = mainPackagePath()
	}
	return Loc{
		Package:  pkg,
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

var pcToLoc sync.Map

func getMsgLogLoc(msg Msg) Loc {
	var pc [1]uintptr
	msg.Callers(1, pc[:])
	locIf, ok := pcToLoc.Load(pc[0])
	if ok {
		return locIf.(Loc)
	}
	loc := locFromPc(pc[0])
	pcToLoc.Store(pc[0], loc)
	return loc
}
