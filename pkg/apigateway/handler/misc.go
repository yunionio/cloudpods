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
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func FetchSession(ctx context.Context, r *http.Request, apiVersion string) *mcclient.ClientSession {
	token := AppContextToken(ctx)
	session := auth.GetSession(ctx, token, FetchRegion(r), apiVersion)
	return session
}

type MiscHandler struct {
	prefix string
}

func NewMiscHandler(prefix string) *MiscHandler {
	return &MiscHandler{prefix}
}

func (h *MiscHandler) GetPrefix() string {
	return h.prefix
}

func (h *MiscHandler) Bind(app *appsrv.Application) {
	prefix := h.prefix

	uploader := UploadHandlerInfo(POST, prefix+"uploads", FetchAuthToken(h.PostUploads))
	app.AddHandler3(uploader)
	app.AddHandler(GET, prefix+"downloads/<template_id>", FetchAuthToken(h.getDownloadsHandler))
}

func UploadHandlerInfo(method, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request)) *appsrv.SHandlerInfo {
	hi := appsrv.SHandlerInfo{}
	hi.SetMethod(method)
	hi.SetPath(prefix)
	hi.SetHandler(handler)
	hi.SetProcessTimeout(6 * time.Hour)
	hi.SetWorkerManager(GetUploaderWorker())
	return &hi
}

func (mh *MiscHandler) PostUploads(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// 10 MB
	var maxMemory int64 = 10 << 20
	e := req.ParseMultipartForm(maxMemory)
	if e != nil {
		httperrors.InvalidInputError(w, "invalid form")
		return
	}

	params := req.MultipartForm.Value

	actions, ok := params["action"]
	if !ok || len(actions) == 0 || len(actions[0]) == 0 {
		err := httperrors.NewInputParameterError("Missing parameter %s", "action")
		httperrors.JsonClientError(w, err)
		return
	}

	switch actions[0] {
	// 主机批量注册
	case "BatchHostRegister":
		mh.DoBatchHostRegister(ctx, w, req)
		return
	default:
		err := httperrors.NewInputParameterError("Unsupported action %s", actions[0])
		httperrors.JsonClientError(w, err)
		return
	}
}

func (mh *MiscHandler) DoBatchHostRegister(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	files := req.MultipartForm.File

	hostfiles, ok := files["hosts"]
	if !ok || len(hostfiles) == 0 || hostfiles[0] == nil {
		e := httperrors.NewInputParameterError("Missing parameter %s", "hosts")
		httperrors.JsonClientError(w, e)
		return
	}

	fileHeader := hostfiles[0].Header
	contentType := fileHeader.Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		e := httperrors.NewInputParameterError("Wrong content type %s, required application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contentType)
		httperrors.JsonClientError(w, e)
		return
	}

	file, err := hostfiles[0].Open()
	defer file.Close()
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't open file")
		httperrors.JsonClientError(w, e)
		return
	}

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't parse file")
		httperrors.JsonClientError(w, e)
	}

	h := ""
	// skipped header row
	for _, row := range xlsx.GetRows("hosts")[1:] {
		if len(row) < 4 {
			log.Debugf("batchHostRegister row length too short (less than 4) %s", row)
		}

		h = h + strings.Join(row, ",") + "\n"
	}

	s := FetchSession(ctx, req, "")
	params := jsonutils.NewDict()
	params.Set("hosts", jsonutils.NewString(h))
	resp, err := modules.Hosts.DoBatchRegister(s, params)
	if err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.JsonClientError(w, e)
		return
	}

	appsrv.SendJSON(w, resp)
}

func (mh *MiscHandler) getDownloadsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	template, ok := params["<template_id>"]
	if !ok || len(template) == 0 {
		httperrors.InvalidInputError(w, "not found")
		return
	}

	switch template {
	case "BatchHostRegister":
		records := [][]string{{"MAC地址", "名称", "IPMI地址", "IPMI用户名", "IPMI密码"}}
		content, err := writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "Attachment; filename=hosts_template.xlsx")
		w.Write(content.Bytes())
		return
	default:
		httperrors.InputParameterError(w, "template not found %s", template)
		return
	}
}

func writeCsv(records [][]string) (bytes.Buffer, error) {
	var content bytes.Buffer
	content.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM, 防止office打开后中文乱码
	writer := csv.NewWriter(&content)
	writer.WriteAll(records)
	if err := writer.Error(); err != nil {
		log.Errorf("error writing csv:%s", err.Error())
		return content, err
	}

	return content, nil
}

func writeXlsx(sheetName string, records [][]string) (bytes.Buffer, error) {
	var content bytes.Buffer
	xlsx := excelize.NewFile()
	xlsx.SetSheetName("Sheet1", sheetName)
	index := xlsx.GetSheetIndex(sheetName)
	for i, record := range records {
		xlsx.SetSheetRow(sheetName, fmt.Sprintf("A%d", i+1), &record)
	}
	xlsx.SetActiveSheet(index)
	if err := xlsx.Write(&content); err != nil {
		log.Errorf("error writing xlsx:%s", err.Error())
		return content, err
	}
	return content, nil
}
