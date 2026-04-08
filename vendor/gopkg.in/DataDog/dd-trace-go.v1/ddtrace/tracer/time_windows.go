// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"time"

	"golang.org/x/sys/windows"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

// This method is more precise than the go1.8 time.Now on Windows
// See https://msdn.microsoft.com/en-us/library/windows/desktop/hh706895(v=vs.85).aspx
// It is however ~10x slower and requires Windows 8+.
func highPrecisionNow() int64 {
	var ft windows.Filetime
	windows.GetSystemTimePreciseAsFileTime(&ft)
	return ft.Nanoseconds()
}

func lowPrecisionNow() int64 {
	return time.Now().UnixNano()
}

// We use this method of initializing now over an init function due to dependency issues. The init
// function may run after other declarations, such as that in payload_test:19, which results in a
// nil dereference panic.
var now func() int64 = func() func() int64 {
	if err := windows.LoadGetSystemTimePreciseAsFileTime(); err != nil {
		log.Warn("Unable to load high precison timer, defaulting to time.Now()")
		return lowPrecisionNow
	} else {
		return highPrecisionNow
	}
}()

var nowTime func() time.Time = func() func() time.Time {
	if err := windows.LoadGetSystemTimePreciseAsFileTime(); err != nil {
		log.Warn("Unable to load high precison timer, defaulting to time.Now()")
		return func() time.Time { return time.Unix(0, lowPrecisionNow()) }
	} else {
		return func() time.Time { return time.Unix(0, highPrecisionNow()) }
	}
}()
