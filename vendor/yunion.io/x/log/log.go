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

// Just logrus wrapper, maybe replace in the future
package log

import (
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
	"yunion.io/x/log/hooks"
)

var (
	verbosity     int32          = 0
	verbosityLock sync.Mutex     = sync.Mutex{}
	logger        *logrus.Logger = logrus.StandardLogger()
)

func init() {
	level := logrus.DebugLevel
	AddHookFormatter(logger)
	SetLogLevel(logger, level)
}

func Logger() *logrus.Logger {
	return logger
}

func SetVerboseLevel(level int32) {
	verbosityLock.Lock()
	defer verbosityLock.Unlock()
	verbosity = level
}

// Verbose is a boolean type that implements Infof (like Printf) etc.
type Verbose bool

// V reports whether verbosity at the call site is at least the requested level.
// The returned value is a boolean of type Verbose, which implements Infof, and Printf etc.
// These methods will write to the Info log if called.
// Thus, one may write either
//	if log.V(2) { log.Infof("log this") }
// or
//	log.V(2).Infof("log this")
// The second form is shorter but the first is cheaper if logging is off because it does
// not evaluate its arguments.
//
// Whether an individual call to V generates a log record depends on the setting of
// the --log-level flags; both are off by default. If the level in the call to
// V is at least the value of --log-level for the source file containing the
// call, the V call will log.
func V(level int32) Verbose {
	if verbosity >= level {
		return Verbose(true)
	}
	return Verbose(false)
}

func (v Verbose) Debugf(format string, args ...interface{}) {
	if v {
		logrus.Debugf(format, args...)
	}
}

func (v Verbose) Printf(format string, args ...interface{}) {
	if v {
		logrus.Printf(format, args...)
	}
}

func (v Verbose) Infof(format string, args ...interface{}) {
	if v {
		logrus.Infof(format, args...)
	}
}

func (v Verbose) Warningf(format string, args ...interface{}) {
	if v {
		logrus.Warningf(format, args...)
	}
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Debugln(args ...interface{}) {
	logrus.Debugln(args...)
}

func Printf(format string, args ...interface{}) {
	logrus.Printf(format, args...)
}

func Println(args ...interface{}) {
	logrus.Println(args...)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func Infoln(args ...interface{}) {
	logrus.Infoln(args...)
}

func Warningf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Warningln(args ...interface{}) {
	logrus.Warnln(args...)
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Errorln(args ...interface{}) {
	logrus.Errorln(args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

func Fatalln(args ...interface{}) {
	logrus.Fatalln(args...)
}

func DisableColors() {
	formatter, ok := logger.Formatter.(*TextFormatter)
	if ok {
		formatter.DisableColors = true
	}
}

func AddHookFormatter(logger *logrus.Logger) {
	pcs := make([]uintptr, 1)
	npcs := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:npcs])
	var myName string
	for {
		f, more := frames.Next()
		myName = f.Function
		if !more {
			break
		}
	}
	logrusPackage := hooks.GetPackageName(myName)
	logger.Hooks.Add(&hooks.CallerHook{logrusPackage})
	logger.Formatter = &TextFormatter{
		TimestampFormat: "060102 15:04:05",
		SpacePadding:    0,
	}
}

func SetLogLevelByString(logger *logrus.Logger, lvl string) error {
	level, err := logrus.ParseLevel(lvl)
	if err != nil {
		return err
	}
	SetLogLevel(logger, level)
	return nil
}

func SetLogLevel(logger *logrus.Logger, level logrus.Level) {
	logger.Level = level
}
