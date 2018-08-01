package appsrv

import (
	"testing"
	"time"
)

func TestWorkerManager(t *testing.T) {
	startTime := time.Now()
	end := make(chan int)
	wm := NewWorkerManager("testwm", 2, 10)
	counter := 0
	for i := 0; i < 10; i += 1 {
		wm.Run(func() {
			counter += 1
			time.Sleep(1 * time.Second)
			if counter >= i {
				end <- 1
			}
		}, nil)
	}
	<-end
	if time.Since(startTime) < 5*time.Second {
		t.Error("Increct timing")
	}
}

func TestWorkerManagerError(t *testing.T) {
	wm := NewWorkerManager("testwm", 2, 10)
	err := make(chan interface{})
	wm.Run(func() {
		panic("Panic inside worker")
	}, err)
	e := WaitChannel(err)
	if e == nil {
		t.Error("Panic not captured")
	}
	err = make(chan interface{})
	wm.Run(func() {
		time.Sleep(1 * time.Second)
	}, err)
	e = WaitChannel(err)
	if e != nil {
		t.Error("Should no error")
	}
}
