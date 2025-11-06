//go:build go1.21
// +build go1.21

package sync

import (
	"sync"
)

var OnceFunc = sync.OnceFunc
