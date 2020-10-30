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
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
)

func SendNoContent(w http.ResponseWriter) {
	w.WriteHeader(204)
	sendBytes(w, []byte{})
}

func Send(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	sendBytes(w, []byte(text))
}

func SendHTML(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	sendBytes(w, []byte(text))
}

func sendBytes(w http.ResponseWriter, output []byte) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(output)), 10))
	w.Write(output)
}

func SendStruct(w http.ResponseWriter, obj interface{}) {
	jsonObj := jsonutils.Marshal(obj)
	SendJSON(w, jsonObj)
}

func SendJSON(w http.ResponseWriter, obj jsonutils.JSONObject) {
	var output []byte
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	if obj != nil {
		output = []byte(obj.String())
	}
	sendBytes(w, output)
}

func SendHeader(w http.ResponseWriter, hdr http.Header) {
	for k, v := range hdr {
		if len(v) > 0 && len(v[0]) > 0 {
			w.Header().Set(k, v[0])
		}
	}
	w.WriteHeader(204)
	w.Write([]byte{})
}

func SendXml(w http.ResponseWriter, hdr http.Header, obj interface{}) {
	SendXmlWithIndent(w, hdr, obj, false)
}

func SendXmlWithIndent(w http.ResponseWriter, hdr http.Header, obj interface{}, indent bool) {
	if !gotypes.IsNil(obj) {
		var xmlBytes []byte
		var err error
		if indent {
			xmlBytes, err = xml.MarshalIndent(obj, "", "  ")
		} else {
			xmlBytes, err = xml.Marshal(obj)
		}
		if err == nil {
			for k, v := range hdr {
				if k != "Content-Type" && k != "Content-Length" {
					w.Header().Set(k, v[0])
				}
			}
			w.Header().Set("Content-Type", "application/xml;charset=utf-8")
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(xmlBytes)+len(xml.Header)), 10))
			w.Write([]byte(xml.Header))
			w.Write(xmlBytes)
		} else {
			w.WriteHeader(400)
			Send(w, err.Error())
		}
	} else {
		for k, v := range hdr {
			if k != "Content-Type" && k != "Content-Length" {
				w.Header().Set(k, v[0])
			}
		}
		SendNoContent(w)
	}
}

func SendStream(w http.ResponseWriter, isPartial bool, hdr http.Header, stream io.ReadCloser, sizeBytes int64) error {
	defer stream.Close()
	if isPartial {
		log.Debugf("send partial 206")
		w.WriteHeader(206)
	} else {
		log.Debugf("send full 200")
		w.WriteHeader(200)
	}
	for k, v := range hdr {
		if k != "Content-Length" {
			log.Debugf("send %s %s", k, v)
			w.Header().Set(k, v[0])
		}
	}
	if sizeBytes > 0 {
		log.Debugf("send content-length %d", sizeBytes)
		w.Header().Set("Content-Length", strconv.FormatInt(sizeBytes, 10))
	}
	offset := 0
	buf := make([]byte, 4096)
	for sizeBytes <= 0 || int64(offset) < sizeBytes {
		n, err := stream.Read(buf)
		if n > 0 {
			woff := 0
			for woff < n {
				m, err := w.Write(buf[woff:n])
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("w.Write read_offset %d write_offset %d", offset, woff))
				}
				woff += m
			}
			offset += n
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "stream.Read")
		}
	}
	return nil
}

func SendRedirect(w http.ResponseWriter, redirectUrl string) {
	w.Header().Set("Location", redirectUrl)
	w.WriteHeader(301)
	w.Write([]byte{})
}

func DisableClientCache(w http.ResponseWriter) {
	// disable client cache
	// Expires: Tue, 03 Jul 2001 06:00:00 GMT
	// Last-Modified: {now} GMT
	// Cache-Control: max-age=0, no-cache, must-revalidate, proxy-revalidate
	w.Header().Set("Expires", "Tue, 03 Jul 2001 06:00:00 GMT")
	cacheSince := time.Now().Format(http.TimeFormat)
	w.Header().Set("Last-Modified", cacheSince)
	w.Header().Set("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
}
