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

package appsrv

import (
	"encoding/json"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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
