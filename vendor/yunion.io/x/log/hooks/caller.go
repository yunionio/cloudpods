// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hooks

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

const callerDelim = "log"

type CallerHook struct {
	UpHookPakcageName string
}

func isInvokeStep(fun, upfun string) bool {
	if strings.Contains(fun, callerDelim) && !strings.Contains(upfun, callerDelim) {
		return true
	}
	return false
}

func (hook *CallerHook) GetCallFields() string {
	var depth int
	for i := 10; i <= 10; i++ {
		pc, _, _, ok := runtime.Caller(i)
		pcUp, _, _, ok := runtime.Caller(i + 1)

		if !ok {
			return "???"
		}

		funcName := runtime.FuncForPC(pc).Name()
		upFuncName := runtime.FuncForPC(pcUp).Name()
		// fmt.Printf("===debug: %d->%q: fname: %q, upName: %q\n", i, i+1, funcName, upFuncName)
		if isInvokeStep(funcName, upFuncName) {
			depth = i + 1
			break
		}
	}

	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "???"
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = funcName[strings.LastIndex(funcName, "/")+1:]
	file = file[strings.LastIndex(file, "/")+1:]
	return fmt.Sprintf("%v(%v:%v)", funcName, file, line)
}

var (
	// Used for caller information initialisation
	callerInitOnce sync.Once
	// logrusPackage  string

	knownLogrusFrames  int = 9
	minimumCallerDepth int = knownLogrusFrames
	maximumCallerDepth int = 25
)

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func GetPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}

func (hook *CallerHook) getCaller() string {
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	var meetUpLogPacket bool
	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := GetPackageName(f.Function)
		if !meetUpLogPacket {
			if pkg == hook.UpHookPakcageName {
				meetUpLogPacket = true
			}
		} else {
			funcName := f.Function[strings.LastIndex(f.Function, "/")+1:]
			file := f.File[strings.LastIndex(f.File, "/")+1:]
			return fmt.Sprintf("%v(%v:%v)", funcName, file, f.Line)
		}
	}
	return "???"
}

func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	entry.Data["caller"] = hook.getCaller()
	return nil
}

func (hook *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
