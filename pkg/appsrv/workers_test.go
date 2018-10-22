package appsrv

import (
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
	err := make(chan interface{})
	wm.Run(func() {
		panic("Panic inside worker")
	}, nil, err)
	e := WaitChannel(err)
	if e == nil {
		t.Error("Panic not captured")
	}
	err = make(chan interface{})
	wm.Run(func() {
		time.Sleep(1 * time.Second)
	}, nil, err)
	e = WaitChannel(err)
	if e != nil {
		t.Error("Should no error")
	}

}
