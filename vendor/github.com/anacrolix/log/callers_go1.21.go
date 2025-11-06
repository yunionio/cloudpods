//go:build go1.21

package log

import (
	"runtime/debug"
	"sync"
)

// Reads the build info to get the true full import path for the main package.
var mainPackagePath = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return info.Path
	}
	return mainPackageFrameImport
})
