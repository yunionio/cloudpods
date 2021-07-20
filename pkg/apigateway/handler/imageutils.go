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
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func readImageForm(r *multipart.Reader) (map[string]string, *multipart.Part, error) {
	params := make(map[string]string)

	maxValueBytes := int64(10 << 20)
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		name := p.FormName()
		if name == "" {
			continue
		}
		filename := p.FileName()

		var b bytes.Buffer

		_, hasContentTypeHeader := p.Header["Content-Type"]
		if !hasContentTypeHeader && filename == "" {
			// value, store as string in memory
			n, err := io.CopyN(&b, p, maxValueBytes+1)
			if err != nil && err != io.EOF {
				return nil, nil, err
			}
			maxValueBytes -= n
			if maxValueBytes < 0 {
				return nil, nil, multipart.ErrMessageTooLarge
			}
			params[name] = b.String()
			continue
		}

		if name == "image" || name == "file" {
			return params, p, nil
		} else {
			return nil, nil, fmt.Errorf("no file uploaded")
		}
	}
	return nil, nil, fmt.Errorf("empty form")
}

func imageUploadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	const (
		invalidForm = "invalid form"
	)
	reader, e := r.MultipartReader()
	if e != nil {
		httperrors.InvalidInputError(ctx, w, invalidForm)
		return
	}

	p, f, e := readImageForm(reader)
	if e != nil {
		httperrors.InvalidInputError(ctx, w, invalidForm)
		return
	}

	params := jsonutils.NewDict()

	name, ok := p["name"]
	if !ok {
		httperrors.InvalidInputError(ctx, w, "missing image name")
		return
	}
	params.Add(jsonutils.NewString(name), "name")

	_imageSize, ok := p["image_size"]
	if !ok {
		httperrors.InvalidInputError(ctx, w, "missing image size")
		return
	}
	imageSize, e := strconv.ParseInt(_imageSize, 10, 64)
	if e != nil {
		httperrors.InvalidInputError(ctx, w, "invalid image size")
		return
	}

	// add all other params
	for k, v := range p {
		if k == "name" || k == "image_size" {
			continue
		}
		params.Add(jsonutils.NewString(v), k)
	}

	token := AppContextToken(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(r), "")

	res, e := modules.Images.Upload(s, params, f, imageSize)
	if e != nil {
		httperrors.GeneralServerError(ctx, w, e)
		return
	} else {
		appsrv.SendJSON(w, res)
	}
}

func uploadHandlerInfo(method, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request)) *appsrv.SHandlerInfo {
	log.Debugf("%s - %s", method, prefix)
	hi := appsrv.SHandlerInfo{}
	hi.SetMethod(method)
	hi.SetPath(prefix)
	hi.SetHandler(handler)
	hi.SetProcessTimeout(6 * time.Hour)
	hi.SetWorkerManager(GetUploaderWorker())
	return &hi
}
