package appsrv

import (
	"context"
	"net/http"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type responseWriterResponse struct {
	count int
	err   error
}

type responseWriterChannel struct {
	backend    http.ResponseWriter
	bodyChan   chan []byte
	bodyResp   chan responseWriterResponse
	statusChan chan int
	statusResp chan bool
}

func newResponseWriterChannel(backend http.ResponseWriter) responseWriterChannel {
	return responseWriterChannel{backend: backend,
		bodyChan:   make(chan []byte),
		bodyResp:   make(chan responseWriterResponse),
		statusChan: make(chan int),
		statusResp: make(chan bool)}
}

func (w *responseWriterChannel) Header() http.Header {
	return w.backend.Header()
}

func (w *responseWriterChannel) Write(bytes []byte) (int, error) {
	w.bodyChan <- bytes
	v := <-w.bodyResp
	return v.count, v.err
}

func (w *responseWriterChannel) WriteHeader(status int) {
	w.statusChan <- status
	<-w.statusResp
}

func (w *responseWriterChannel) wait(ctx context.Context, errChan chan interface{}) interface{} {
	var err interface{}
	stop := false
	for !stop {
		select {
		case <-ctx.Done():
			// ctx deadline reached, timeout
			err = httperrors.NewTimeoutError("request process timeout")
			stop = true
		case e, more := <-errChan:
			if more {
				err = e
			} else {
				stop = true
			}
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
	close(w.bodyChan)
	close(w.bodyResp)
	close(w.statusChan)
	close(w.statusResp)
}
