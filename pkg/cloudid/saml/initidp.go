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

package saml

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/cloudid/saml/providers"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func initSAMLIdp(app *appsrv.Application, prefix string) error {
	spFunc := func(ctx context.Context, idpId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
		token := auth.FetchUserCredential(ctx, nil)
		log.Debugf("Recive SP initiated Login: %s", sp.GetEntityId())
		data := samlutils.SSAMLSpInitiatedLoginData{}
		driver := providers.FindDriver(sp.GetEntityId())
		if driver == nil {
			return data, errors.Wrapf(httperrors.ErrResourceNotFound, "entityID %s not found", sp.GetEntityId())
		}
		data, err := driver.GetSpInitiatedLoginData(idpId, token.GetUserId(), sp)
		if err != nil {
			return data, errors.Wrap(err, "driver.GetSpInitiatedLoginData")
		}
		return data, nil
		/*
			switch sp.GetEntityId() {
			case SAML_ENTITY_ID_HUAWEI_CLOUD: // 华为云 SSO

			case SAML_ENTITY_ID_TENCENT_CLOUD: // 腾讯云 role SSO

			}
			return data
		*/
	}

	idpFunc := func(ctx context.Context, sp *idp.SSAMLServiceProvider, idpId string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
		token := auth.FetchUserCredential(ctx, nil)
		log.Debugf("Recive IDP initiated Login: %s", sp.GetEntityId())
		data := samlutils.SSAMLIdpInitiatedLoginData{}
		driver := providers.FindDriver(sp.GetEntityId())
		if driver == nil {
			return data, errors.Wrapf(httperrors.ErrResourceNotFound, "entityID %s not found", sp.GetEntityId())
		}
		data, err := driver.GetIdpInitiatedLoginData(idpId, token.GetUserId(), sp)
		if err != nil {
			return data, errors.Wrap(err, "driver.GetIdpInitiatedLoginData")
		}
		return data, nil
	}

	logoutFunc := func(ctx context.Context, idpId string) string {
		return fmt.Sprintf(`<!DOCTYPE html><html lang="zh_CN"><head><meta charset="utf-8"><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body><h1>成功退出登录，<a href="%s">重新登录</a></h1></body></html>`, options.Options.ApiServer)
	}

	idpInst := idp.NewIdpInstance(saml, spFunc, idpFunc, logoutFunc)
	for entityId, drvFactory := range providers.AllDrivers() {
		filePath := path.Join(options.Options.CloudSAMLMetadataPath, drvFactory.GetMetadataFilename())
		metaBytes, err := ioutil.ReadFile(filePath)
		if err != nil || len(metaBytes) == 0 {
			metaUrl := drvFactory.GetMetadataUrl()
			if len(metaUrl) > 0 {
				log.Debugf("[%s] metadata file load failed, try download from %s", entityId, metaUrl)
				httpcli := httputils.GetDefaultClient()
				resp, err := httpcli.Get(metaUrl)
				if err != nil {
					return errors.Wrapf(err, "http get %s fail", metaUrl)
				}
				metaBytes, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					return errors.Wrapf(err, "read body %s fail", metaUrl)
				}
			} else {
				return errors.Wrapf(err, "read file %s fail", filePath)
			}
		}
		err = idpInst.AddSPMetadata(metaBytes)
		if err != nil {
			return errors.Wrapf(err, "AddSPMetadata %s", metaBytes)
		}
	}

	idpInst.AddHandlers(app, prefix, auth.Authenticate)
	idpInst.SetHtmlTemplate(`<!DOCTYPE html><html lang="zh_CN"><head><meta charset="utf-8"><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body><h1>正在跳转到云控制台，请等待。。。</h1>$FORM$</body></html>`)

	idpInstance = idpInst

	return nil
}
