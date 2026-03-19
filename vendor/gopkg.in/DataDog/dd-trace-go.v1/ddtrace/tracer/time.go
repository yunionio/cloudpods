// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build !windows
// +build !windows

package tracer

import "time"

// nowTime returns the current time, as computed by Time.Now().
var nowTime func() time.Time = func() time.Time { return time.Now() }

// now returns the current UNIX time in nanoseconds, as computed by Time.UnixNano().
var now func() int64 = func() int64 { return time.Now().UnixNano() }
