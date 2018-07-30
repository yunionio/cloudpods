package appsrv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yunionio/pkg/util/version"
)

type FilterHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request)

func VersionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, version.GetShortString())
}

func PingHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

/*
func CORSHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    reqHdrs, enableCors := r.Header["Access-Control-Request-Headers"]
    if enableCors {
        w.Header().Set("Access-Control-Allow-Origin", getRequestOrigin(r))
        allowHdrs := strings.Join(reqHdrs, ",")
        allowHdrs = fmt.Sprintf("%s,%s", allowHdrs, "Authorization")
        w.Header().Set("Vary", "Origin,Access-Control-Request-Method,Access-Control-Request-Headers")
        w.Header().Set("Access-Control-Allow-Headers", allowHdrs)
        w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,PATCH,DELETE")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Expose-Headers", allowHdrs)
        w.Header().Set("Access-Control-Max-Age", "86400")
    }
}*/
