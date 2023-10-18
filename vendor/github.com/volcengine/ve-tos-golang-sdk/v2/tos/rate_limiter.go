package tos

import (
	"sync"
	"time"
)

const (
	minRate     = 1024
	minCapacity = 10 * 1024
)

type defaultRateLimit struct {
	rate     int64
	capacity int64

	currentAmount int64
	sync.Mutex
	lastConsumeTime time.Time
}

func NewDefaultRateLimit(rate int64, capacity int64) RateLimiter {

	if rate < minRate {
		rate = minRate
	}
	if capacity < minCapacity {
		capacity = minCapacity
	}
	return &defaultRateLimit{
		rate:            rate,
		capacity:        capacity,
		lastConsumeTime: time.Now(),
		currentAmount:   capacity,
		Mutex:           sync.Mutex{},
	}
}

func (d *defaultRateLimit) Acquire(want int64) (ok bool, timeToWait time.Duration) {
	d.Lock()
	defer d.Unlock()
	if want > d.capacity {
		want = d.capacity
	}
	increment := int64(time.Since(d.lastConsumeTime).Seconds() * float64(d.rate))
	if increment+d.currentAmount > d.capacity {
		d.currentAmount = d.capacity
	} else {
		d.currentAmount += increment
	}
	if want > d.currentAmount {
		timeToWaitSec := float64(want-d.currentAmount) / float64(d.rate)
		return false, time.Duration(timeToWaitSec * float64(time.Second))
	}
	d.lastConsumeTime = time.Now()
	d.currentAmount -= want
	return true, 0
}
