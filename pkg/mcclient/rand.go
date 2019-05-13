package mcclient

import (
	"math/rand"
	"time"
)

func init() {
	// initialize random seed
	rand.Seed(time.Now().UnixNano())
}
