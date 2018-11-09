package main

import (
	"context"
	"net/http"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
)

func main() {
	app := appsrv.NewApplication("test", 4)
	app.AddHandler("GET", "/delay", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 20)
		log.Debugf("end of delay sleep....")
		appsrv.Send(w, "pong")
	})
	app.AddHandler("GET", "/panic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("the handler is panic")
	})
	app.AddHandler("GET", "/delaypanic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 1)
		panic("the handler is panic")
	})
	app.ListenAndServe("0.0.0.0:44444")
}
