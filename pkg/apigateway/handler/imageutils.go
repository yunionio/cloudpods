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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

var IMAGE_DOWNLOAD_PUBLIC_KEY = stringutils.UUID4()

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
	s := auth.GetSession(ctx, token, FetchRegion(r))

	res, e := modules.Images.Upload(s, params, f, imageSize)
	if e != nil {
		httperrors.GeneralServerError(ctx, w, e)
		return
	} else {
		appsrv.SendJSON(w, res)
	}
}

func imageDownloadValidateStatus(s *mcclient.ClientSession, id string, format string) error {
	var image jsonutils.JSONObject
	var err error
	if len(format) == 0 {
		image, err = modules.Images.Get(s, id, nil)
		if err != nil {
			return errors.Wrap(err, "images.get")
		}
	} else {
		resp, e := modules.Images.GetSpecific(s, id, "subformats", nil)
		if e != nil {
			return errors.Wrap(e, "images.get.subformats")
		}

		images, _ := resp.(*jsonutils.JSONArray).GetArray()
		for i := range images {
			if f, _ := images[i].GetString("format"); f == format {
				image = images[i]
			}
		}
	}

	status, _ := image.GetString("status")
	if status != "active" {
		return httperrors.NewInvalidStatusError("image is not in status 'active'")
	}

	return nil
}

func imageDownloadUrl(id string, format string) (jsonutils.JSONObject, error) {
	// 加密下载url
	expired := time.Now().Add(24 * time.Hour)
	imageInfo := jsonutils.NewDict()
	imageInfo.Set("id", jsonutils.NewString(id))
	imageInfo.Set("format", jsonutils.NewString(format))
	imageInfo.Set("expired", jsonutils.NewInt(expired.Unix()))
	token, err := utils.EncryptAESBase64Url(IMAGE_DOWNLOAD_PUBLIC_KEY, imageInfo.String())
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Set("signature", jsonutils.NewString(token))
	return ret, nil
}

func imageDownload(ctx context.Context, w http.ResponseWriter, s *mcclient.ClientSession, id string, format string) {
	meta, body, size, err := modules.Images.Download2(s, id, format, false)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	name, _ := meta.GetString("name")
	if len(name) == 0 {
		format, _ := meta.GetString("disk_format")
		name = "os_image"
		if len(format) > 0 {
			name += "." + format
		}
	}

	hdr := http.Header{}
	hdr.Set("Content-Type", "application/octet-stream")
	hdr.Set("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", name))
	appsrv.SendStream(w, false, hdr, body, size)
	return
}

func imageDownloadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	token := AppContextToken(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(r))
	// input params
	imageId, ok := params["<image_id>"]
	if !ok || len(imageId) == 0 {
		httperrors.MissingParameterError(ctx, w, "image_id")
		return
	}
	// 是否直接下载
	format, _ := query.GetString("format")
	err := imageDownloadValidateStatus(s, imageId, format)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	direct, _ := query.Bool("direct")
	if !direct {
		ret, err := imageDownloadUrl(imageId, format)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}

		appsrv.SendJSON(w, ret)
		return
	}

	// 直接下载镜像
	imageDownload(ctx, w, s, imageId, format)
}

func imageDownloadByUrlHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	// input params
	token, err := query.GetString("signature")
	if len(token) == 0 {
		log.Debugf("get signature %s", err)
		httperrors.MissingParameterError(ctx, w, "signature")
		return
	}

	data, err := utils.DescryptAESBase64Url(IMAGE_DOWNLOAD_PUBLIC_KEY, token)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "invalid download token")
		return
	}

	d, _ := jsonutils.ParseString(data)
	id, _ := d.GetString("id")
	if err != nil || len(id) == 0 {
		httperrors.InputParameterError(ctx, w, "invalid download token")
		return
	}

	expired, _ := d.Int("expired")
	if time.Now().Unix() > expired {
		httperrors.BadRequestError(ctx, w, "image download url is expired")
		return
	}
	format, _ := d.GetString("format")
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	imageDownload(ctx, w, s, id, format)
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
