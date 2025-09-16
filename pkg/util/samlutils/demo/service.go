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

package demo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
	"yunion.io/x/onecloud/pkg/util/samlutils/sp"
)

type Options struct {
	Help    bool     `help:"show help"`
	Cert    string   `help:"certificate file"`
	Key     string   `help:"certificate private key file"`
	Port    int      `help:"listening port"`
	Entity  string   `help:"SAML entityID"`
	IdpId   string   `help:"IDP ID"`
	SpMeta  []string `help:"ServiceProvider metadata filename"`
	IdpMeta []string `help:"IdentityProvider metadata filename"`
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func StartServer() {
	err := prepareServer()
	if err != nil {
		showErrorAndExit(err)
	} else {
		fmt.Println("exit cleanly")
	}
}

func prepareServer() error {
	parser, err := structarg.NewArgumentParser(
		&Options{},
		"samldemo",
		"A demo SAML 2.0 https server",
		`See "ipmicli help COMMAND" for help on a specific command.`,
	)
	if err != nil {
		return errors.Wrap(err, "NewArgumentParser")
	}

	err = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*Options)

	if options.Help {
		fmt.Print(parser.HelpString())
		return nil
	}

	if len(options.Entity) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "empty entityID")
	}
	if options.Port <= 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "port must be positive integer")
	}
	if len(options.Key) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "key file must be present")
	}
	if !fileutils2.Exists(options.Key) {
		return errors.Wrapf(httperrors.ErrInputParameter, "key %s not found", options.Key)
	}
	if len(options.Cert) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "cert file must be present")
	}
	if !fileutils2.Exists(options.Cert) {
		return errors.Wrapf(httperrors.ErrInputParameter, "cert %s not found", options.Cert)
	}

	app := appsrv.NewApplication("samldemo", 4, false)

	saml, err := samlutils.NewSAMLInstance(options.Entity, options.Cert, options.Key)
	if err != nil {
		return errors.Wrap(err, "NewSAMLInstance")
	}

	spFunc := func(ctx context.Context, idpId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
		log.Debugf("Recive SP initiated Login: %s", sp.GetEntityId())
		data := samlutils.SSAMLSpInitiatedLoginData{}
		switch sp.GetEntityId() {
		case "https://auth.huaweicloud.com/": // 华为云 SSO
			data.NameId = "yunionoss"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
			data.AudienceRestriction = sp.GetEntityId()
			for k, v := range map[string]string{
				// "xUserId":    "052d45a3e70010440f92c000d9e3f260",
				// "xAccountId": "052d45a3e70010440f92c000d9e3f260",
				// "bpId":       "c58a60a2e0a046c8afa77286924c2b0d",
				// "name":       "yunionoss",
				// "email":      "qiujian@yunion.cn",
				// "mobile":     "13811299225",
				"User":  "ec2admin",
				"Group": "ec2admin",
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name: k, FriendlyName: k,
					NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
					Values:     []string{v},
				})
			}
		case "https://samltest.id/saml/sp": // samltest.id SSO
			data.NameId = "yunion"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
			data.AudienceRestriction = sp.GetEntityId()
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:         "urn:oid:0.9.2342.19200300.100.1.1",
					friendlyName: "uid",
					value:        "9646D89D-F5E7-F0E4-C545A9B2F4B7956B",
				},
				{
					name:         "urn:oid:0.9.2342.19200300.100.1.3",
					friendlyName: "mail",
					value:        "samltest@yunion.io",
				},
				{
					name:         "urn:oid:2.5.4.4",
					friendlyName: "sn",
					value:        "Jian",
				},
				{
					name:         "urn:oid:2.5.4.42",
					friendlyName: "givenName",
					value:        "Jian",
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
					Values:       []string{v.value},
				})
			}
		case "cloud.tencent.com": // 腾讯云 role SSO
			data.NameId = "cvmcosreadonly"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
			data.AudienceRestriction = "https://cloud.tencent.com"
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:         "https://cloud.tencent.com/SAML/Attributes/Role",
					friendlyName: "RoleEntitlement",
					value:        "qcs::cam::uin/100008182714:roleName/cvmcosreadonly,qcs::cam::uin/100008182714:saml-provider/saml.yunion.io",
				},
				{
					name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
					friendlyName: "RoleSessionName",
					value:        "cvmcosreadonly",
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
		case "urn:federation:MicrosoftOnline":
			data.NameId = sp.Username
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
			data.AudienceRestriction = sp.GetEntityId()
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
					value: data.NameId,
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
			return data, nil
		case "google.com/a/yunion-hk.com":
			data.NameId = "qiujian"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
			data.AudienceRestriction = sp.GetEntityId()
			for k, v := range map[string]string{
				"user.email": "qiujian@yunion-hk.com",
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name: k, FriendlyName: k,
					NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
					Values:     []string{v},
				})
			}
		case "google.com":
			data.NameId = "qiujian@yunion-hk.com"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_EMAIL
			data.AudienceRestriction = sp.GetEntityId()
			for k, v := range map[string]string{
				"user.email": "qiujian@yunion-hk.com",
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name: k, FriendlyName: k,
					NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
					Values:     []string{v},
				})
			}
		}
		return data, nil
	}

	idpFunc := func(ctx context.Context, sp *idp.SSAMLServiceProvider, idpId, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
		log.Debugf("Recive IDP initiated Login: %s", sp.GetEntityId())
		data := samlutils.SSAMLIdpInitiatedLoginData{}
		switch sp.GetEntityId() {
		case "urn:alibaba:cloudcomputing": // 阿里云role SSO
			data.NameId = "ecsossreadonly"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
			data.AudienceRestriction = sp.GetEntityId()
			for k, v := range map[string]string{
				"https://www.aliyun.com/SAML-Role/Attributes/Role":            "acs:ram::1123247935774897:role/administrator,acs:ram::1123247935774897:saml-provider/saml.yunion.io",
				"https://www.aliyun.com/SAML-Role/Attributes/RoleSessionName": "ecsossreadonly",
				"https://www.aliyun.com/SAML-Role/Attributes/SessionDuration": "1800",
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:   k,
					Values: []string{v},
				})
			}
			data.RelayState = "https://homenew.console.aliyun.com/"
		case "urn:amazon:webservices:cn-north-1": // AWS CN role SSO
			data.NameId = "ec2s3readonly"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
			data.AudienceRestriction = "https://signin.amazonaws.cn/saml"
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:         "https://aws.amazon.com/SAML/Attributes/Role",
					friendlyName: "RoleEntitlement",
					value:        "arn:aws-cn:iam::248697896586:role/ec2s3readonly,arn:aws-cn:iam::248697896586:saml-provider/saml.yunion.io",
				},
				{
					name:         "https://aws.amazon.com/SAML/Attributes/RoleSessionName",
					friendlyName: "RoleSessionName",
					value:        "ec2s3readonly",
				},
				{
					name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.3",
					friendlyName: "eduPersonOrgDN",
					value:        "ec2s3readonly",
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
			data.RelayState = "https://console.amazonaws.cn/"
		case "urn:amazon:webservices": // AWS Global role SSO
			data.NameId = "ec2s3readonly"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
			data.AudienceRestriction = "https://signin.aws.amazon.com/saml"
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:         "https://aws.amazon.com/SAML/Attributes/Role",
					friendlyName: "RoleEntitlement",
					value:        "arn:aws:iam::285906155448:role/ec2s3readonly,arn:aws:iam::285906155448:saml-provider/saml.yunion.cn",
				},
				{
					name:         "https://aws.amazon.com/SAML/Attributes/RoleSessionName",
					friendlyName: "RoleSessionName",
					value:        "ec2s3readonly",
				},
				{
					name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.3",
					friendlyName: "eduPersonOrgDN",
					value:        "ec2s3readonly",
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
			data.RelayState = "https://console.aws.amazon.com/"
		case "cloud.tencent.com": // 腾讯云 role SSO
			data.NameId = "cvmcosreadonly"
			data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
			data.AudienceRestriction = "https://cloud.tencent.com"
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:         "https://cloud.tencent.com/SAML/Attributes/Role",
					friendlyName: "RoleEntitlement",
					value:        "qcs::cam::uin/100008182714:roleName/cvmcosreadonly,qcs::cam::uin/100008182714:saml-provider/saml.yunion.io",
				},
				{
					name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
					friendlyName: "RoleSessionName",
					value:        "cvmcosreadonly",
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
			data.RelayState = "https://console.cloud.tencent.com/"
		}
		return data, nil
	}

	logoutFunc := func(ctx context.Context, idpId string) string {
		return fmt.Sprintf(`<!DOCTYPE html><html lang="zh_CN"><head><meta charset="utf-8"><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body><h1>成功退出登录，<a href="%s">重新登录</a></h1></body></html>`, httputils.JoinPath(options.Entity, "SAML/idp"))
	}

	idpInst := idp.NewIdpInstance(saml, spFunc, idpFunc, logoutFunc)
	for _, spMetaFile := range options.SpMeta {
		err := idpInst.AddSPMetadataFile(spMetaFile)
		if err != nil {
			return errors.Wrapf(err, "AddSPMetadataFile %s", spMetaFile)
		}
	}
	idpInst.AddHandlers(app, "SAML/idp", nil)
	idpInst.SetHtmlTemplate(i18n.NewTableEntry().CN(`<!DOCTYPE html><html lang="zh_CN"><head><meta charset="utf-8"><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body><h1>正在跳转到云控制台，请等待。。。</h1>$FORM$</body></html>`))

	app.AddHandler("GET", "SAML/idp", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		idpInitUrl := httputils.JoinPath(options.Entity, "SAML/idp/sso")

		htmlBuf := strings.Builder{}
		htmlBuf.WriteString(`<!doctype html><html lang=en><body><ol>`)
		// IDP initiated
		for _, v := range []struct {
			name     string
			entityID string
		}{
			{
				name:     "Aliyun Role SSO",
				entityID: "urn:alibaba:cloudcomputing",
			},
			{
				name:     "AWS CN Role SSO",
				entityID: "urn:amazon:webservices:cn-north-1",
			},
			{
				name:     "AWS Global Role SSO",
				entityID: "urn:amazon:webservices",
			},
			{
				name:     "Tencent Cloud Role SSO",
				entityID: "cloud.tencent.com",
			},
		} {
			query := samlutils.SIdpInitiatedLoginInput{
				EntityID: v.entityID,
				IdpId:    options.IdpId,
			}
			htmlBuf.WriteString(fmt.Sprintf(`<li><a href="%s?%s">%s (IDP-Initiated)</a></li>`, idpInitUrl, jsonutils.Marshal(query).QueryString(), v.name))
		}

		for _, v := range []struct {
			name string
			url  string
		}{
			/*{
				name: "Huawei cloud partner SSO",
				url:  "https://auth.huaweicloud.com/authui/saml/login?xAccountType=yunion_IDP&isFirstLogin=false&service=https%3a%2f%2fconsole.huaweicloud.com%2fiam%2f",
			},*/
			{
				name: "Huawei cloud SSO",
				url:  "https://auth.huaweicloud.com/authui/federation/websso?domain_id=052d45a3e70010440f92c000d9e3f260&idp=yunion&protocol=saml",
			},
			{
				name: "Tencent cloud SSO",
				url:  "https://cloud.tencent.com/login/forwardIdp/100008182714/saml.yunion.io",
			},
			{
				name: "Google cloud SSO",
				url:  "https://www.google.com/a/yunion-hk.com/ServiceLogin?continue=https://console.cloud.google.com",
			},
			{
				name: "Azure cloud SSO",
				url:  "https://login.microsoftonline.com/redeem?rd=https%3a%2f%2finvitations.microsoft.com%2fredeem%2f%3ftenant%3d17493ddf-fa90-4f95-8576-5df011c126e5%26user%3d3bc1c055-aa14-4795-aef0-5970b00d03c7%26ticket%3d0GDu%252bZ7nLbg01rYL5u%252b401%252bOLyZjxPewSBJIAZZ7E0U%253d%26ver%3d2.0",
			},
		} {
			htmlBuf.WriteString(fmt.Sprintf(`<li><a href="%s">%s (SP-Initiated)</a></li>`, v.url, v.name))
		}

		htmlBuf.WriteString(`</ol></body></html>`)
		appsrv.SendHTML(w, htmlBuf.String())
	})

	consumeFunc := func(ctx context.Context, w http.ResponseWriter, idp *sp.SSAMLIdentityProvider, result sp.SSAMLAssertionConsumeResult) error {
		html := strings.Builder{}
		html.WriteString("<!doctype html><html lang=en><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\"></head><body><ol>")
		html.WriteString(fmt.Sprintf("<li>RequestId: %s</li>", result.RequestID))
		html.WriteString(fmt.Sprintf("<li>RelayState: %s</li>", result.RelayState))
		for _, v := range result.Attributes {
			html.WriteString(fmt.Sprintf("<li>%s(%s): %s</li>", v.Name, v.FriendlyName, v.Values))
		}
		html.WriteString("</ol></body></html>")
		appsrv.SendHTML(w, html.String())
		return nil
	}

	spLoginFunc := func(ctx context.Context, idp *sp.SSAMLIdentityProvider) (sp.SSAMLSpInitiatedLoginRequest, error) {
		result := sp.SSAMLSpInitiatedLoginRequest{}
		result.RequestID = samlutils.GenerateSAMLId()
		return result, nil
	}

	spInst := sp.NewSpInstance(saml, "Yunion SAML Demo Service", consumeFunc, spLoginFunc)
	for _, idpFile := range options.IdpMeta {
		err := spInst.AddIdpMetadataFile(idpFile)
		if err != nil {
			return errors.Wrap(err, "AddIdpMetadataFile")
		}
	}
	spInst.AddHandlers(app, "SAML/sp")

	app.AddHandler("GET", "SAML/sp", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		spInitUrl := httputils.JoinPath(options.Entity, "SAML/sp/sso")

		htmlBuf := strings.Builder{}
		htmlBuf.WriteString(`<!doctype html><html lang=en><body><ol>`)

		for _, idp := range spInst.GetIdentityProviders() {
			entityId := idp.GetEntityId()
			htmlBuf.WriteString(fmt.Sprintf(`<li><a href="%s?EntityID=%s">%s (SP-Initiated)</a></li>`, spInitUrl, url.QueryEscape(entityId), entityId))
		}

		htmlBuf.WriteString(`</ol></body></html>`)
		appsrv.SendHTML(w, htmlBuf.String())
	})

	addr := fmt.Sprintf(":%d", options.Port)
	app.ListenAndServeTLS(addr, options.Cert, options.Key)

	return nil
}
