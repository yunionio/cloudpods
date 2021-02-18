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

package utils

import (
	"context"
	"fmt"
	"time"
)

// FibonacciRetryFunc is the type of func retrier calls.
type FibonacciRetryFunc func(retrier FibonacciRetrier) (done bool, err error)

// FibonacciRetrierErrorType is the error type that could be returned by
// retrier
type FibonacciRetrierErrorType int

const (
	FibonacciRetrierErrorMaxTriesExceeded FibonacciRetrierErrorType = iota
	FibonacciRetrierErrorMaxElapseExceeded
)

// FibonacciRetrierError is the type of error retrier returns when MaxTries or
// MaxElapse are exceeded.
type FibonacciRetrierError struct {
	Type FibonacciRetrierErrorType
	// Fibr is a copy of the original retrier.
	Fibr FibonacciRetrier
	// Err is the original error returned by retry func.  It can be nil.
	Err error
}

func (err FibonacciRetrierError) Error() string {
	switch err.Type {
	case FibonacciRetrierErrorMaxTriesExceeded:
		return fmt.Sprintf("tried %d times, exceeding %d, last err %v", err.Fibr.Tried(), err.Fibr.MaxTries, err.Err)
	case FibonacciRetrierErrorMaxElapseExceeded:
		return fmt.Sprintf("elapsed time %s, exceeding %s, last err %v", err.Fibr.Elapsed(), err.Fibr.MaxElapse, err.Err)
	default:
		return fmt.Sprintf("unexpected retrier err type: %#v", err)
	}
}

// matchFibonacciRetrierErrorType returns true if err is of type
// FibonacciRetrierError and the type is same as typ.
func matchFibonacciRetrierErrorType(err error, typ FibonacciRetrierErrorType) (bool, FibonacciRetrierError) {
	fibrErr, ok := err.(FibonacciRetrierError)
	if !ok {
		return false, FibonacciRetrierError{}
	}
	if fibrErr.Type != typ {
		return false, fibrErr
	}
	return true, fibrErr
}

// FibonacciRetrier calls RetryFunc until it returns done, or waits in
// fibonacci way until MaxTries, MaxElapse are exceeded whichever happens first
type FibonacciRetrier struct {
	// T0 is the 1st item in fibonacci series
	T0 time.Duration
	// T1 is the 2nd item in fibonacci series
	T1        time.Duration
	MaxTries  int
	MaxElapse time.Duration
	RetryFunc FibonacciRetryFunc

	startTime time.Time
	endTime   time.Time
	tried     int
}

// Start initiates the call and wait sequence
//
// If RetryFunc returns with done being true.  err will also be what RetryFunc returns
//
// Otherwise Start will return with done being false and err of type
// FibonacciRetrierError with last err returned by RetryFunc wrapped in, or
// ctx.Err() if it's done
func (fibr *FibonacciRetrier) Start(ctx context.Context) (done bool, err error) {
	fibr.tried = 0
	fibr.startTime = time.Now()
	defer func() {
		fibr.endTime = time.Now()
	}()
	for {
		done, err = fibr.RetryFunc(*fibr)
		if done {
			return
		}
		fibr.tried += 1
		if fibr.MaxTries > 0 && fibr.tried >= fibr.MaxTries {
			return done, fibr.newError(FibonacciRetrierErrorMaxTriesExceeded, err)
		}
		fibr.T0, fibr.T1 = fibr.T1, fibr.T0+fibr.T1
		// allow it to exceed in one round
		if fibr.MaxElapse > 0 && fibr.Elapsed() > fibr.MaxElapse {
			return done, fibr.newError(FibonacciRetrierErrorMaxElapseExceeded, err)
		}
		// maxTries	1	2	3	4	5	6	7
		// T0			1	2	3	5	8	13
		// Elapse	0	1	3	6	11	19	32
		select {
		case <-time.After(fibr.T0):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

// Tries returns number of times RetryFunc has been called, excluding the
// current executing one
func (fibr *FibonacciRetrier) Tried() int {
	return fibr.tried
}

// Elapsed returns the duration the retrier has run
func (fibr *FibonacciRetrier) Elapsed() time.Duration {
	if fibr.startTime.IsZero() {
		return time.Duration(0)
	} else if !fibr.endTime.IsZero() {
		return fibr.endTime.Sub(fibr.startTime)
	} else {
		return time.Since(fibr.startTime)
	}
}

func (fibr *FibonacciRetrier) newError(typ FibonacciRetrierErrorType, origErr error) error {
	err := FibonacciRetrierError{
		Err:  origErr,
		Fibr: *fibr,
		Type: typ,
	}
	return err
}

// NewFibonacciRetrierMaxTries returnes a retrier that tries at most maxTries
// times with the first 2 items being 1 second
func NewFibonacciRetrierMaxTries(maxTries int, retryFunc FibonacciRetryFunc) *FibonacciRetrier {
	fibr := &FibonacciRetrier{
		T0:        time.Second,
		T1:        time.Second,
		MaxTries:  maxTries,
		RetryFunc: retryFunc,
	}
	return fibr
}

// NewFibonacciRetrierMaxTries returnes a retrier that tries for at most
// maxElapse duration with the first 2 items being 1 second
func NewFibonacciRetrierMaxElapse(maxElapse time.Duration, retryFunc FibonacciRetryFunc) *FibonacciRetrier {
	fibr := &FibonacciRetrier{
		T0:        time.Second,
		T1:        time.Second,
		MaxElapse: maxElapse,
		RetryFunc: retryFunc,
	}
	return fibr
}
