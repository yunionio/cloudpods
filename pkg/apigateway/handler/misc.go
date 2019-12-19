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
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	HOST_MAC                    = "*MAC地址"
	HOST_NAME                   = "*名称"
	HOST_IPMI_ADDR              = "*IPMI地址"
	HOST_IPMI_USERNAME          = "*IPMI用户名"
	HOST_IPMI_PASSWORD          = "*IPMI密码"
	HOST_MNG_IP_ADDR            = "*管理口IP地址"
	HOST_IPMI_ADDR_OPTIONAL     = "IPMI地址"
	HOST_IPMI_USERNAME_OPTIONAL = "IPMI用户名"
	HOST_IPMI_PASSWORD_OPTIONAL = "IPMI密码"
	HOST_MNG_IP_ADDR_OPTIONAL   = "管理口IP地址"
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
	app.AddHandler(POST, prefix+"piuploads", FetchAuthToken(h.postPIUploads)) // itsm process instances upload api
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
	// 用户批量注册
	case "BatchUserRegister":
		mh.DoBatchUserRegister(ctx, w, req)
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
		return
	}

	rows := xlsx.GetRows("hosts")
	if len(rows) == 0 {
		e := httperrors.NewGeneralError(fmt.Errorf("empty file content"))
		httperrors.JsonClientError(w, e)
		return
	}

	h := ""
	paramKeys := []string{}
	for _, title := range rows[0] {
		switch title {
		case HOST_MAC:
			paramKeys = append(paramKeys, "access_mac")
		case HOST_NAME:
			paramKeys = append(paramKeys, "name")
		case HOST_IPMI_ADDR, HOST_IPMI_ADDR_OPTIONAL:
			paramKeys = append(paramKeys, "ipmi_ip_addr")
		case HOST_IPMI_USERNAME, HOST_IPMI_USERNAME_OPTIONAL:
			paramKeys = append(paramKeys, "ipmi_username")
		case HOST_IPMI_PASSWORD, HOST_IPMI_PASSWORD_OPTIONAL:
			paramKeys = append(paramKeys, "ipmi_password")
		case HOST_MNG_IP_ADDR, HOST_MNG_IP_ADDR_OPTIONAL:
			paramKeys = append(paramKeys, "access_ip")
		default:
			e := httperrors.NewInternalServerError("empty file content")
			httperrors.JsonClientError(w, e)
			return
		}
	}

	// skipped header row
	for _, row := range xlsx.GetRows("hosts")[1:] {
		h = h + strings.Join(row, ",") + "\n"
	}

	s := FetchSession(ctx, req, "")
	params := jsonutils.NewDict()
	params.Set("hosts", jsonutils.NewString(h))
	resp, err := modules.Hosts.DoBatchRegister(s, paramKeys, params)
	if err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.JsonClientError(w, e)
		return
	}

	appsrv.SendJSON(w, resp)
}

func (mh *MiscHandler) DoBatchUserRegister(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	s := FetchSession(ctx, req, "")
	files := req.MultipartForm.File

	userfiles, ok := files["users"]
	if !ok || len(userfiles) == 0 || userfiles[0] == nil {
		e := httperrors.NewInputParameterError("Missing parameter %s", "users")
		httperrors.JsonClientError(w, e)
		return
	}

	fileHeader := userfiles[0].Header
	contentType := fileHeader.Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		e := httperrors.NewInputParameterError("Wrong content type %s, required application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contentType)
		httperrors.JsonClientError(w, e)
		return
	}

	file, err := userfiles[0].Open()
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
		return
	}

	// skipped header row
	rows := xlsx.GetRows("users")
	if len(rows) <= 1 {
		e := httperrors.NewInputParameterError("empty file")
		httperrors.JsonClientError(w, e)
		return
	}

	users := []jsonutils.JSONObject{}
	names := map[string]bool{}
	domains := map[string]string{}
	for i, row := range rows[1:] {
		name := row[0]
		if len(name) == 0 {
			e := httperrors.NewClientError("row %d name is empty", i+2)
			httperrors.JsonClientError(w, e)
			return
		}

		if _, ok := names[name]; ok {
			e := httperrors.NewClientError("duplicate name %s", row[0])
			httperrors.JsonClientError(w, e)
			return
		} else {
			names[name] = true
			_, err := modules.UsersV3.Get(s, name, nil)
			if err == nil {
				continue
			}
		}

		domainId, ok := domains[row[1]]
		if !ok {
			id, err := modules.Domains.GetId(s, row[1], nil)
			if err != nil {
				httperrors.JsonClientError(w, httperrors.NewGeneralError(err))
				return
			}

			domainId = id
			domains[row[1]] = id
		}

		user := jsonutils.NewDict()
		user.Add(jsonutils.NewString(name), "name")
		user.Add(jsonutils.NewString(domainId), "domain_id")
		user.Add(jsonutils.NewString("OneCloud@2019"), "password")
		if strings.ToLower(row[2]) == "true" || strings.ToLower(row[2]) == "1" {
			user.Add(jsonutils.JSONTrue, "allow_web_console")
		} else {
			user.Add(jsonutils.JSONFalse, "allow_web_console")
		}

		users = append(users, user)
	}

	// batch create
	var userG errgroup.Group
	for i := range users {
		user := users[i]
		userG.Go(func() error {
			_, err := modules.UsersV3.Create(s, user)
			return err
		})
	}

	if err := userG.Wait(); err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.GeneralServerError(w, e)
		return
	}

	appsrv.SendJSON(w, jsonutils.NewDict())
}

func (mh *MiscHandler) getDownloadsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	template, ok := params["<template_id>"]
	if !ok || len(template) == 0 {
		httperrors.InvalidInputError(w, "not found")
		return
	}

	var err error
	var content bytes.Buffer
	switch template {
	case "BatchHostRegister":
		records := [][]string{{HOST_MAC, HOST_NAME, HOST_IPMI_ADDR_OPTIONAL, HOST_IPMI_USERNAME_OPTIONAL, HOST_IPMI_PASSWORD_OPTIONAL}}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}
	case "BatchHostISORegister":
		records := [][]string{{HOST_NAME, HOST_IPMI_ADDR, HOST_IPMI_USERNAME, HOST_IPMI_PASSWORD, HOST_MNG_IP_ADDR}}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}
	case "BatchHostPXERegister":
		records := [][]string{{HOST_NAME, HOST_IPMI_ADDR, HOST_IPMI_USERNAME, HOST_IPMI_PASSWORD, HOST_MNG_IP_ADDR_OPTIONAL}}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}
	case "BatchUserRegister":
		records := [][]string{{"用户名（user）", "部门/域（domain）", "是否登录控制台（allow_web_console：true、false）"}}
		content, err = writeXlsx("users", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}
	case "BatchProjectRegister":
		var titles []string
		if options.Options.NonDefaultDomainProjects {
			titles = []string{"项目名称", "域", "配额"}
		} else {
			titles = []string{"项目名称", "域"}
		}

		records := [][]string{titles}
		content, err = writeXlsx("projects", records)
		if err != nil {
			httperrors.InternalServerError(w, "internal server error")
			return
		}
	default:
		httperrors.InputParameterError(w, "template not found %s", template)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "Attachment; filename=template.xlsx")
	w.Write(content.Bytes())
	return
}

func (mh *MiscHandler) postPIUploads(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// 5 MB
	var maxMemory int64 = 5 << 20
	if req.ContentLength > maxMemory {
		httperrors.InvalidInputError(w, "request body is too large.")
		return
	}

	if !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
		httperrors.InvalidInputError(w, "invalid multipart form")
		return
	}

	s := FetchSession(ctx, req, "")
	header := http.Header{}
	header.Set("Content-Type", req.Header.Get("Content-Type"))
	header.Set("Content-Length", req.Header.Get("Content-Length"))
	resp, err := modules.ProcessInstance.Upload(s, header, req.Body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	appsrv.SendJSON(w, resp)
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
