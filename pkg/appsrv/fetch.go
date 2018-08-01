package appsrv

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/yunionio/jsonutils"
)

func Fetch(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	return ioutil.ReadAll(req.Body)
}

func FetchStruct(req *http.Request, v interface{}) error {
	b, e := Fetch(req)
	if e != nil {
		return e
	}
	if len(b) > 0 {
		return json.Unmarshal(b, v)
	} else {
		return nil
	}
}

func FetchJSON(req *http.Request) (jsonutils.JSONObject, error) {
	b, e := Fetch(req)
	if e != nil {
		return nil, e
	}
	if len(b) > 0 {
		return jsonutils.Parse(b)
	} else {
		return nil, nil
	}
}
