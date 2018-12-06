package appsrv

import (
	"context"
	"net/http"

	"fmt"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type responseWriterResponse struct {
	count int
	err   error
}

type responseWriterChannel struct {
	backend http.ResponseWriter

	bodyChan   chan []byte
	bodyResp   chan responseWriterResponse
	statusChan chan int
	statusResp chan bool

	isClosed bool
}

func newResponseWriterChannel(backend http.ResponseWriter) responseWriterChannel {
	return responseWriterChannel{
		backend:    backend,
		bodyChan:   make(chan []byte),
		bodyResp:   make(chan responseWriterResponse),
		statusChan: make(chan int),
		statusResp: make(chan bool),
		isClosed:   false,
	}
}

func (w *responseWriterChannel) Header() http.Header {
	return w.backend.Header()
}

func (w *responseWriterChannel) Write(bytes []byte) (int, error) {
	if w.isClosed {
		return 0, fmt.Errorf("response stream has been closed")
	}
	w.bodyChan <- bytes
	v := <-w.bodyResp
	return v.count, v.err
}

func (w *responseWriterChannel) WriteHeader(status int) {
	if w.isClosed {
		return
	}
	w.statusChan <- status
	<-w.statusResp
}

// implent http.Flusher
func (w *responseWriterChannel) Flush() {
	if w.isClosed {
		return
	}
	if f, ok := w.backend.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *responseWriterChannel) wait(ctx context.Context, workerChan chan *SWorker) interface{} {
	var err error
	var worker *SWorker
	stop := false
	for !stop {
		select {
		case worker = <-workerChan:
		case <-ctx.Done():
			// ctx deadline reached, timeout
			if worker != nil {
				worker.Detach("timeout")
			}
			err = httperrors.NewTimeoutError("request process timeout")
			stop = true
		case bytes, more := <-w.bodyChan:
			// log.Print("Recive body ", len(bytes), " more ", more)
			if more {
				c, e := w.backend.Write(bytes)
				w.bodyResp <- responseWriterResponse{count: c, err: e}
			} else {
				stop = true
			}
		case status, more := <-w.statusChan:
			// log.Print("Recive status ", status, " more ", more)
			if more {
				w.backend.WriteHeader(status)
				w.statusResp <- true
			} else {
				stop = true
			}
		}
	}
	return err
}

func (w *responseWriterChannel) closeChannels() {
	if w.isClosed {
		return
	}
	w.isClosed = true

	close(w.bodyChan)
	close(w.bodyResp)
	close(w.statusChan)
	close(w.statusResp)
}
