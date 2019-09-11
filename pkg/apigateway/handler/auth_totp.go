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
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/skip2/go-qrcode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

// 转换成base64编码qrcode。
// https://docs.openstack.org/keystone/rocky/advanced-topics/auth-totp.html
// otpauth://totp/{name}?secret={secret}&issuer={issuer}
// https://authenticator.ppl.family/
func toQrcode(secret string, token mcclient.TokenCredential) (string, error) {
	_secret := base32.StdEncoding.EncodeToString([]byte(secret))
	// https://github.com/google/google-authenticator/wiki/Key-Uri-Format
	// issuer Onecloud.Domain
	issuer := "Onecloud"
	if len(token.GetDomainName()) > 0 {
		issuer = fmt.Sprintf("Onecloud.%s", token.GetDomainName())
		issuer = url.PathEscape(issuer)
	}

	uri := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", issuer, token.GetUserName(), _secret, issuer)

	c, err := qrcode.Encode(uri, qrcode.High, 256)
	if err != nil {
		log.Errorf(err.Error())
		return "", httperrors.NewInternalServerError("generate totp qrcode failed")
	}

	return base64.StdEncoding.EncodeToString(c), nil
}

// 检查用户是否设置了TOTP credential.true -- 已设置；false--未设置
func isUserTotpCredInitialed(s *mcclient.ClientSession, uid string) (bool, error) {
	_, err := modules.Credentials.GetTotpSecret(s, uid)
	if err == nil {
		return true, nil
	}

	switch e := err.(type) {
	case *httputils.JSONClientError:
		if e.Code == 404 {
			return false, nil
		}
		return false, err
	default:
		return false, err
	}
}

// 创建用户TOTP credential.并返回携带认证信息的Qrcode（base64编码，png格式）。
func doCreateUserTotpCred(s *mcclient.ClientSession, token mcclient.TokenCredential) (string, error) {
	secret, err := modules.Credentials.CreateTotpSecret(s, token.GetUserId())
	if err != nil {
		return "", err
	}

	return toQrcode(secret, token)
}

// 初始化用户credential.如果新创建则返回携带认证信息的Qrcode（base64编码，png格式）。否则返回空字符串
func initializeUserTotpCred(s *mcclient.ClientSession, token mcclient.TokenCredential) (string, error) {
	uid := token.GetUserId()
	if len(uid) == 0 {
		return "", httperrors.NewConflictError("uid is empty")
	}

	ok, err := isUserTotpCredInitialed(s, uid)
	if err != nil {
		return "", err
	}

	// 已经创建返回空字符串.否则返回携带认证信息的Qrcode（base64编码，png格式）
	if ok {
		return "", nil
	}

	return doCreateUserTotpCred(s, token)
}

// 重置totp credential.检查用户密码找回问题答案是否正确。如果正确则进行密码重置操作.返回Qrcode
func resetUserTotpCred(s *mcclient.ClientSession, token mcclient.TokenCredential) (string, error) {
	uid := token.GetUserId()
	if len(uid) == 0 {
		return "", httperrors.NewConflictError("uid is empty")
	}

	ok, err := isUserTotpCredInitialed(s, uid)
	if err != nil {
		return "", err
	}

	// 如果已经设置密钥则删除旧认证信息
	if ok {
		err := modules.Credentials.RemoveTotpSecrets(s, uid)
		if err != nil {
			return "", err
		}
	}

	return doCreateUserTotpCred(s, token)
}

// 检查用户是否设置了TOTP 密码恢复问题.true -- 已设置；false--未设置
func isUserTotpRecoverySecretsInitialed(s *mcclient.ClientSession, uid string) (bool, error) {
	_, err := modules.Credentials.GetRecoverySecrets(s, uid)
	if err == nil {
		return true, nil
	}

	switch e := err.(type) {
	case *httputils.JSONClientError:
		if e.Code == 404 {
			return false, nil
		}
		return false, err
	default:
		return false, err
	}
}

// 设置重置密码问题.
func setTotpRecoverySecrets(s *mcclient.ClientSession, uid string, questions []modules.SRecoverySecret) error {
	exists, err := isUserTotpRecoverySecretsInitialed(s, uid)
	if err != nil {
		return err
	}

	if exists {
		err := modules.Credentials.RemoveRecoverySecrets(s, uid)
		if err != nil {
			return err
		}
	}

	return modules.Credentials.SaveRecoverySecrets(s, uid, questions)
}

// 验证重置密码问题.验证未通过则返回错误提示
func validateTotpRecoverySecrets(s *mcclient.ClientSession, uid string, questions jsonutils.JSONObject) error {
	_qs := make([]modules.SRecoverySecret, 0)
	err := questions.Unmarshal(&_qs)
	if err != nil {
		return httperrors.NewInputParameterError("input parameter error")
	}

	qs := map[string]string{}
	for _, q := range _qs {
		qs[q.Question] = q.Answer
	}

	ss, err := modules.Credentials.GetRecoverySecrets(s, uid)
	if err != nil {
		return err
	}

	if len(ss) == 0 {
		return httperrors.NewConflictError("TOTP recovery questions do not exist")
	}

	for _, s := range ss {
		if v, existis := qs[s.Question]; !existis {
			return httperrors.NewInputParameterError("questions not found")
		} else {
			if v != s.Answer {
				return httperrors.NewInputParameterError("%s answer is incorrect", s.Question)
			}
		}
	}

	return nil
}

// 验证OTP
func ValidatePasscodeHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := SetAuthToken(ctx, w, req)
	if err != nil {
		httperrors.InvalidCredentialError(w, "set auth token: %v", err)
		return
	}

	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	_, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}

	passcode, err := body.GetString("passcode")
	if err != nil {
		httperrors.MissingParameterError(w, "passcode")
		return
	}

	if len(passcode) != 6 {
		httperrors.InputParameterError(w, "passcode is a 6-digits string")
		return
	}

	tid := getAuthToken(req)
	totp := clientman.TokenMan.GetTotp(tid)
	err = totp.VerifyTotpPasscode(s, t.GetUserId(), passcode)
	if err != nil {
		log.Warningf("VerifyTotpPasscode %s", err.Error())
		httperrors.InvalidCredentialError(w, "invalid passcode: %v", err)
		return
	} else {
		clientman.TokenMan.SaveTotp(tid)
	}

	appsrv.SendJSON(w, jsonutils.NewDict())
}

// 验证OTP credential重置问题.如果答案正确，返回重置后的Qrcode（base64编码，png格式）。
func ResetTotpSecrets(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := SetAuthToken(ctx, w, req)
	if err != nil {
		httperrors.InvalidCredentialError(w, "set auth token: %v", err)
		return
	}

	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	_, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}

	uid := t.GetUserId()
	if len(uid) == 0 {
		httperrors.ConflictError(w, "uid is empty")
		return
	}

	err = validateTotpRecoverySecrets(s, uid, body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	code, err := resetUserTotpCred(s, t)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	resp := jsonutils.NewDict()
	resp.Add(jsonutils.NewString(code), "qrcode")
	appsrv.SendJSON(w, resp)
}

// 获取OTP 重置密码问题列表。
func ListTotpRecoveryQuestions(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := SetAuthToken(ctx, w, req)
	if err != nil {
		httperrors.InvalidCredentialError(w, "set auth token: %v", err)
		return
	}

	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	// 做缓存？
	ss, err := modules.Credentials.GetRecoverySecrets(s, t.GetUserId())
	if len(ss) == 0 {
		log.Errorf("ListTotpRecoveryQuestions %s", err.Error())
		httperrors.NotFoundError(w, "no revocery questions.")
		return
	}

	resp := jsonutils.NewDict()
	questions := jsonutils.NewArray()
	for _, s := range ss {
		questions.Add(jsonutils.NewString(s.Question))
	}
	resp.Add(questions, "data")
	appsrv.SendJSON(w, resp)
}

// 提交OTP 重置密码问题。
func ResetTotpRecoveryQuestions(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := SetAuthToken(ctx, w, req)
	if err != nil {
		httperrors.InvalidCredentialError(w, "set auth token: %v", err)
		return
	}

	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	_, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}

	questions := make([]modules.SRecoverySecret, 0)
	err = body.Unmarshal(&questions)
	if err != nil {
		httperrors.InvalidInputError(w, "unmarshal questions: %v", err)
		return
	}

	err = setTotpRecoverySecrets(s, t.GetUserId(), questions)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	appsrv.SendJSON(w, jsonutils.NewDict())
}
