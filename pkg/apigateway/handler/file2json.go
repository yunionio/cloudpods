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

package handler

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type File2JsonHandler struct {
	prefix string
}

func NewFile2JsonHandler(prefix string) *File2JsonHandler {
	return &File2JsonHandler{
		prefix: prefix,
	}
}

func (h *File2JsonHandler) Bind(app *appsrv.Application) {
	prefix := h.prefix
	app.AddHandler(POST, prefix+"file2json", FetchAuthToken(h.file2Json))
}

func (h *File2JsonHandler) file2Json(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// 1 Mb
	err := req.ParseMultipartForm(1 << 20)
	if err != nil {
		httperrors.InvalidInputError(w, "invalid form")
		return
	}

	if req.MultipartForm != nil {
		for filename := range req.MultipartForm.File {
			file, _, err := req.FormFile(filename)
			if err != nil {
				httperrors.InvalidInputError(w, "file to open file %s error: %v", filename)
				return
			}
			buf := make([]byte, 1<<20)
			_, err = file.Read(buf)
			if err != nil {
				httperrors.GeneralServerError(w, errors.Wrap(err, "file.Read"))
				return
			}
			body, err := jsonutils.Parse(buf)
			if err != nil {
				httperrors.InvalidInputError(w, "parse json error: %v", err)
				return
			}
			appsrv.SendJSON(w, body)
			return
		}
	}

	httperrors.InvalidInputError(w, "missing file")
}
