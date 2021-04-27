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
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const contentTypeSpreadsheet = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

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
	HOST_MNG_MAC_ADDR_OPTIONAL  = "管理口MAC地址"
)

const (
	BATCH_USER_REGISTER_QUANTITY_LIMITATION = 1000
	BATCH_HOST_REGISTER_QUANTITY_LIMITATION = 1000
)

var (
	BatchHostRegisterTemplate    = []string{HOST_MAC, HOST_NAME, HOST_IPMI_ADDR_OPTIONAL, HOST_IPMI_USERNAME_OPTIONAL, HOST_IPMI_PASSWORD_OPTIONAL}
	BatchHostISORegisterTemplate = []string{HOST_NAME, HOST_IPMI_ADDR, HOST_IPMI_USERNAME, HOST_IPMI_PASSWORD, HOST_MNG_IP_ADDR}
	BatchHostPXERegisterTemplate = []string{HOST_NAME, HOST_IPMI_ADDR, HOST_IPMI_USERNAME, HOST_IPMI_PASSWORD, HOST_MNG_MAC_ADDR_OPTIONAL, HOST_MNG_IP_ADDR_OPTIONAL}
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
	imageUploader := uploadHandlerInfo("POST", prefix+"/imageutils/upload", FetchAuthToken(imageUploadHandler))
	app.AddHandler3(imageUploader)
	s3upload := uploadHandlerInfo(POST, prefix+"s3uploads", FetchAuthToken(h.postS3UploadHandler))
	app.AddHandler3(s3upload)
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
		httperrors.InvalidInputError(ctx, w, "invalid form")
		return
	}

	params := req.MultipartForm.Value

	actions, ok := params["action"]
	if !ok || len(actions) == 0 || len(actions[0]) == 0 {
		err := httperrors.NewInputParameterError("Missing parameter %s", "action")
		httperrors.JsonClientError(ctx, w, err)
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
		httperrors.JsonClientError(ctx, w, err)
		return
	}
}

func (mh *MiscHandler) DoBatchHostRegister(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	files := req.MultipartForm.File

	hostfiles, ok := files["hosts"]
	if !ok || len(hostfiles) == 0 || hostfiles[0] == nil {
		e := httperrors.NewInputParameterError("Missing parameter %s", "hosts")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	fileHeader := hostfiles[0].Header
	contentType := fileHeader.Get("Content-Type")
	if contentType != contentTypeSpreadsheet {
		e := httperrors.NewInputParameterError("Wrong content type %s, want %s", contentType, contentTypeSpreadsheet)
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	file, err := hostfiles[0].Open()
	defer file.Close()
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't open file")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't parse file")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	rows := xlsx.GetRows("hosts")
	if len(rows) == 0 {
		e := httperrors.NewGeneralError(fmt.Errorf("empty file content"))
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	// check header line
	titlesOk := false
	for _, t := range [][]string{BatchHostRegisterTemplate, BatchHostISORegisterTemplate, BatchHostPXERegisterTemplate} {
		if len(t) == len(rows[0]) {
			for _, title := range rows[0] {
				if !utils.IsInStringArray(title, t) {
					break
				}
			}

			titlesOk = true
		}
	}

	if !titlesOk {
		httperrors.InputParameterError(ctx, w, "template file is invalid. please check.")
		return
	}

	paramKeys := []string{}
	i1 := -1
	i2 := -1
	for i, title := range rows[0] {
		switch title {
		case HOST_MAC, HOST_MNG_MAC_ADDR_OPTIONAL:
			paramKeys = append(paramKeys, "access_mac")
		case HOST_NAME:
			paramKeys = append(paramKeys, "name")
		case HOST_IPMI_ADDR, HOST_IPMI_ADDR_OPTIONAL:
			i1 = i
			paramKeys = append(paramKeys, "ipmi_ip_addr")
		case HOST_IPMI_USERNAME, HOST_IPMI_USERNAME_OPTIONAL:
			paramKeys = append(paramKeys, "ipmi_username")
		case HOST_IPMI_PASSWORD, HOST_IPMI_PASSWORD_OPTIONAL:
			paramKeys = append(paramKeys, "ipmi_password")
		case HOST_MNG_IP_ADDR, HOST_MNG_IP_ADDR_OPTIONAL:
			i2 = i
			paramKeys = append(paramKeys, "access_ip")
		default:
			e := httperrors.NewInternalServerError("empty file content")
			httperrors.JsonClientError(ctx, w, e)
			return
		}
	}

	// skipped header row
	if len(rows) > BATCH_HOST_REGISTER_QUANTITY_LIMITATION {
		e := httperrors.NewInputParameterError(fmt.Sprintf("beyond limitation. excel file rows must less than %d", BATCH_HOST_REGISTER_QUANTITY_LIMITATION))
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	ips := []string{}
	hosts := bytes.Buffer{}
	for _, row := range rows[1:] {
		var e *httputils.JSONClientError
		if i1 >= 0 && len(row[i1]) > 0 {
			i1Ip := fmt.Sprintf("%d-%s", i1, row[i1])
			if utils.IsInStringArray(i1Ip, ips) {
				e = httperrors.NewDuplicateIdError("ip", row[i1])
			} else {
				ips = append(ips, i1Ip)
			}
		}

		if i2 >= 0 && len(row[i2]) > 0 {
			i2Ip := fmt.Sprintf("%d-%s", i2, row[i2])
			if utils.IsInStringArray(i2Ip, ips) {
				e = httperrors.NewDuplicateIdError("ip", row[i2])
			} else {
				ips = append(ips, i2Ip)
			}
		}

		if e != nil {
			httperrors.JsonClientError(ctx, w, e)
			return
		}

		hosts.WriteString(strings.Join(row, ",") + "\n")
	}

	params := jsonutils.NewDict()
	s := FetchSession(ctx, req, "")
	params.Set("hosts", jsonutils.NewString(hosts.String()))

	// extra params
	for k, values := range req.MultipartForm.Value {
		if len(values) > 0 && k != "action" {
			params.Set(k, jsonutils.NewString(values[0]))
		}
	}

	submitResult, err := modules.Hosts.BatchRegister(s, paramKeys, params)
	if err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(submitResult))
}

func (mh *MiscHandler) DoBatchUserRegister(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	adminS := auth.GetAdminSession(ctx, FetchRegion(req), "")
	s := FetchSession(ctx, req, "")
	files := req.MultipartForm.File

	userfiles, ok := files["users"]
	if !ok || len(userfiles) == 0 || userfiles[0] == nil {
		e := httperrors.NewInputParameterError("Missing parameter %s", "users")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	fileHeader := userfiles[0].Header
	contentType := fileHeader.Get("Content-Type")
	if contentType != contentTypeSpreadsheet {
		e := httperrors.NewInputParameterError("Wrong content type %s, want %s", contentType, contentTypeSpreadsheet)
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	file, err := userfiles[0].Open()
	defer file.Close()
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't open file")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		log.Errorf(err.Error())
		e := httperrors.NewInternalServerError("can't parse file")
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	// skipped header row
	rows := xlsx.GetRows("users")
	if len(rows) <= 1 {
		e := httperrors.NewInputParameterError("empty file content")
		httperrors.JsonClientError(ctx, w, e)
		return
	} else if len(rows) > BATCH_USER_REGISTER_QUANTITY_LIMITATION {
		e := httperrors.NewInputParameterError(fmt.Sprintf("beyond limitation.excel file rows must less than %d", BATCH_USER_REGISTER_QUANTITY_LIMITATION))
		httperrors.JsonClientError(ctx, w, e)
		return
	}

	users := []jsonutils.JSONObject{}
	names := map[string]bool{}
	domains := map[string]string{}
	for i, row := range rows[1:] {
		rowIdx := i + 2
		name := row[0]
		password := row[1]
		domain := row[2]
		allowWebConsole := strings.ToLower(row[3])

		// 忽略空白行
		rowStr := strings.Join(row, "")
		if len(strings.TrimSpace(rowStr)) == 0 {
			continue
		}

		if len(name) == 0 {
			e := httperrors.NewClientError("row %d name is empty", rowIdx)
			httperrors.JsonClientError(ctx, w, e)
			return
		}

		if len(password) == 0 {
			e := httperrors.NewClientError("row %d password is empty", rowIdx)
			httperrors.JsonClientError(ctx, w, e)
			return
		}

		domainId, ok := domains[domain]
		if !ok {
			if len(domain) == 0 {
				e := httperrors.NewClientError("row %d domain is empty", rowIdx)
				httperrors.JsonClientError(ctx, w, e)
				return
			}

			id, err := modules.Domains.GetId(adminS, domain, nil)
			if err != nil {
				httperrors.JsonClientError(ctx, w, httperrors.NewGeneralError(err))
				return
			}

			domainId = id
			domains[domain] = id
		}

		if _, ok := names[name+"/"+domainId]; ok {
			e := httperrors.NewClientError("row %d duplicate name %s", rowIdx, name)
			httperrors.JsonClientError(ctx, w, e)
			return
		} else {
			names[name+"/"+domainId] = true
			params := jsonutils.NewDict()
			params.Set("domain_id", jsonutils.NewString(domainId))
			_, err := modules.UsersV3.Get(s, name, params)
			if err == nil {
				continue
			}
		}

		user := jsonutils.NewDict()
		user.Add(jsonutils.NewString(name), "name")
		user.Add(jsonutils.NewString(domainId), "domain_id")
		if len(password) > 0 {
			user.Add(jsonutils.NewString(password), "password")
			user.Add(jsonutils.JSONTrue, "skip_password_complexity_check")
		}

		if allowWebConsole == "true" || allowWebConsole == "1" {
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
		httperrors.GeneralServerError(ctx, w, e)
		return
	}

	appsrv.SendJSON(w, jsonutils.NewDict())
}

func (mh *MiscHandler) getDownloadsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	template, ok := params["<template_id>"]
	if !ok || len(template) == 0 {
		httperrors.InvalidInputError(ctx, w, "template_id")
		return
	}

	var err error
	var content bytes.Buffer
	switch template {
	case "BatchHostRegister":
		records := [][]string{BatchHostRegisterTemplate}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(ctx, w, "internal server error")
			return
		}
	case "BatchHostISORegister":
		records := [][]string{BatchHostISORegisterTemplate}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(ctx, w, "internal server error")
			return
		}
	case "BatchHostPXERegister":
		records := [][]string{BatchHostPXERegisterTemplate}
		content, err = writeXlsx("hosts", records)
		if err != nil {
			httperrors.InternalServerError(ctx, w, "internal server error")
			return
		}
	case "BatchUserRegister":
		records := [][]string{{"*用户名（user）", "*用户密码（password）", "*部门/域（domain）", "*是否登录控制台（allow_web_console：true、false）"}}
		content, err = writeXlsx("users", records)
		if err != nil {
			httperrors.InternalServerError(ctx, w, "internal server error")
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
			httperrors.InternalServerError(ctx, w, "internal server error")
			return
		}
	default:
		httperrors.InputParameterError(ctx, w, "template not found %s", template)
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
		httperrors.InvalidInputError(ctx, w, "request body is too large.")
		return
	}

	if !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
		httperrors.InvalidInputError(ctx, w, "invalid multipart form")
		return
	}

	s := FetchSession(ctx, req, "")
	header := http.Header{}
	header.Set("Content-Type", req.Header.Get("Content-Type"))
	header.Set("Content-Length", req.Header.Get("Content-Length"))
	resp, err := modules.ProcessInstance.Upload(s, header, req.Body)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	appsrv.SendJSON(w, resp)
}

func (mh *MiscHandler) postS3UploadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	reader, e := r.MultipartReader()
	if e != nil {
		log.Debugf("postS3UploadHandler.MultipartReader %s", e)
		httperrors.InvalidInputError(ctx, w, "invalid form")
		return
	}

	p, f, e := readImageForm(reader)
	if e != nil {
		log.Debugf("postS3UploadHandler.readImageForm %s", e)
		httperrors.InvalidInputError(ctx, w, "invalid form")
		return
	}

	bucket_id, ok := p["bucket_id"]
	if !ok {
		httperrors.MissingParameterError(ctx, w, "bucket_id")
		return
	}

	key, ok := p["key"]
	if !ok {
		httperrors.MissingParameterError(ctx, w, "key")
		return
	}

	_content_length, ok := p["content_length"]
	if !ok {
		httperrors.MissingParameterError(ctx, w, "content_length")
		return
	}

	content_length, e := strconv.ParseInt(_content_length, 10, 64)
	if e != nil {
		httperrors.InvalidInputError(ctx, w, "invalid content_length %s", _content_length)
		return
	}

	storage_class, _ := p["storage_class"]
	acl, _ := p["acl"]

	token := AppContextToken(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(r), "")

	meta := http.Header{}
	meta.Set("Content-Type", "application/octet-stream")
	e = modules.Buckets.Upload(s, bucket_id, key, f, content_length, storage_class, acl, meta)
	if e != nil {
		httperrors.GeneralServerError(ctx, w, e)
		return
	}
	appsrv.SendJSON(w, jsonutils.NewDict())
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
