package appsrv

import (
	"encoding/json"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
)

func Send(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(text))
}

func SendStruct(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	b, e := json.Marshal(obj)
	if e != nil {
		log.Errorln("SendStruct json Marshal error: ", e)
	}
	w.Write(b)
}

func SendJSON(w http.ResponseWriter, obj jsonutils.JSONObject) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(obj.String()))
}
