package appsrv

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerManager(t *testing.T) {
	enableDebug()
	startTime := time.Now()
	// end := make(chan int)
	wm := NewWorkerManager("testwm", 2, 10)
	counter := 0
	for i := 0; i < 10; i += 1 {
		wm.Run(func() {
			counter += 1
			time.Sleep(1 * time.Second)
		}, nil, nil)
	}
	for wm.ActiveWorkerCount() != 0 {
		time.Sleep(time.Second)
	}
	if time.Since(startTime) < 5*time.Second {
		t.Error("Incorrect timing")
	}
}

func TestWorkerManagerError(t *testing.T) {
	wm := NewWorkerManager("testwm", 2, 10)
	errCbFactory := func(wg *sync.WaitGroup, errMark *bool) func(error) {
		return func(error) {
			defer wg.Done()
			if errMark != nil && !*errMark {
				*errMark = true
			}
		}
	}
	t.Run("normal", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		errMark := false
		errCb := errCbFactory(wg, &errMark)
		wg.Add(1)
		wm.Run(func() {
			defer wg.Done()
		}, nil, errCb)
		wg.Wait()
		if errMark {
			t.Errorf("should be normal")
		}
	})
	t.Run("panic", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		errMark := false
		errCb := errCbFactory(wg, &errMark)
		wg.Add(2) // 1 for errCb
		wm.Run(func() {
			defer wg.Done()
			panic("panic inside worker")
		}, nil, errCb)
		wg.Wait()
		if !errMark {
			t.Errorf("expecting error")
		}
	})
}
