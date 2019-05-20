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

	"github.com/sirupsen/logrus"
)

const callerDelim = "log"

type CallerHook struct{}

func isInvokeStep(fun, upfun string) bool {
	if strings.Contains(fun, callerDelim) && !strings.Contains(upfun, callerDelim) {
		return true
	}
	return false
}

func GetCallFields() string {
	var depth int
	for i := 3; i <= 9; i++ {
		pc, _, _, ok := runtime.Caller(i)
		pcUp, _, _, ok := runtime.Caller(i + 1)

		if !ok {
			return "???"
		}

		funcName := runtime.FuncForPC(pc).Name()
		upFuncName := runtime.FuncForPC(pcUp).Name()
		//fmt.Printf("===debug: %d->%q: fname: %q, upName: %q\n", i, f1, funcName, upFuncName)
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

func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	entry.Data["caller"] = GetCallFields()
	return nil
}

func (hook *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
