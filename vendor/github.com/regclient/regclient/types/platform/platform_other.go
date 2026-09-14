//go:build !windows
// +build !windows

package platform

import "runtime"

// Local retrieves the local platform details
func Local() Platform {
	return Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Variant:      cpuVariant(),
	}
}
