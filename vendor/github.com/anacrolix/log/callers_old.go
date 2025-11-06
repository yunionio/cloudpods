//go:build !go1.21

package log

import (
	"runtime/debug"
	"sync"
)

var (
	// The cached full import path for the true full import path for the main package in the current
	// binary.
	cachedMainPackagePath string
	cacheMainPackagePath  sync.Once
)

// Returns the full import path for the main package in the current binary, through the Once guard.
func mainPackagePath() string {
	cacheMainPackagePath.Do(initMainPackagePath)
	return cachedMainPackagePath
}

func initMainPackagePath() {
	cachedMainPackagePath = getMainPackagePath()
}

// Reads the build info to get the true full import path for the main package.
func getMainPackagePath() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return info.Path
	}
	return mainPackageFrameImport
}
