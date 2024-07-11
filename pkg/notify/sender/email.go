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

package sender

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	gomail "gopkg.in/mail.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type errorMap map[string]error

func (em errorMap) Error() string {
	msg := make(map[string]string)
	for k, e := range em {
		msg[k] = e.Error()
	}
	return jsonutils.Marshal(msg).String()
}

type SEmailSender struct {
	config map[string]api.SNotifyConfigContent
}

func (emailSender *SEmailSender) GetSenderType() string {
	return api.EMAIL
}

func (emailSender *SEmailSender) Send(ctx context.Context, args api.SendParams) error {
	// 初始化emaliClient
	hostNmae, hostPort, userName, password := models.ConfigMap[api.EMAIL].Content.Hostname, models.ConfigMap[api.EMAIL].Content.Hostport, models.ConfigMap[api.EMAIL].Content.Username, models.ConfigMap[api.EMAIL].Content.Password
	dialer := gomail.NewDialer(hostNmae, hostPort, userName, password)

	// 是否支持ssl
	if models.ConfigMap[api.EMAIL].Content.SslGlobal {
		dialer.SSL = true
	} else {
		dialer.SSL = false
		dialer.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	if len(args.EmailMsg.To) > 0 {
		// emailMsg不为空时
		sender, err := dialer.Dial()
		if err != nil {
			return errors.Wrap(err, "dialer.Dial")
		}
		retErr := errorMap{}

		destMap := make(map[string]int)

		for _, tos := range [][]string{
			args.EmailMsg.To,
			args.EmailMsg.Cc,
			args.EmailMsg.Bcc,
		} {
			for _, to := range tos {
				to = strings.ToLower(to)
				if _, ok := destMap[to]; !ok && len(to) > 0 {
					destMap[to] = 1
				}
			}
		}

		for to := range destMap {
			gmsg := gomail.NewMessage()
			gmsg.SetHeader("From", models.ConfigMap[api.EMAIL].Content.SenderAddress)
			gmsg.SetHeader("To", to)
			gmsg.SetHeader("Subject", args.EmailMsg.Subject)
			gmsg.SetBody("text/html", args.EmailMsg.Body)

			for i := range args.EmailMsg.Attachments {
				attach := args.EmailMsg.Attachments[i]
				gmsg.Attach(attach.Filename,
					gomail.SetCopyFunc(func(w io.Writer) error {
						mime := attach.Mime
						if len(mime) == 0 {
							mime = "application/octet-stream"
						}
						_, err := w.Write([]byte("Content-Type: " + attach.Mime))
						return errors.Wrap(err, "WriteMime")
					}),
					gomail.SetHeader(map[string][]string{
						"Content-Disposition": {
							fmt.Sprintf(`attachment; filename="%s"`, mime.QEncoding.Encode("UTF-8", attach.Filename)),
						},
					}),
					gomail.SetCopyFunc(func(w io.Writer) error {
						contBytes, err := base64.StdEncoding.DecodeString(attach.Base64Content)
						if err != nil {
							return errors.Wrap(err, "base64.StdEncoding.DecodeString")
						}
						_, err = w.Write(contBytes)
						return errors.Wrap(err, "WriteContent")
					}),
				)
			}
			errs := make([]error, 0)
			for tryTime := 3; tryTime > 0; tryTime-- {
				err = gomail.Send(sender, gmsg)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "Send"))
					time.Sleep(time.Second * 10)
					continue
				}
				errs = errs[0:0]
				break
			}

			if len(errs) > 0 {
				retErr[to] = errors.NewAggregate(errs)
			}
		}
		if len(retErr) > 0 {
			log.Errorf("send email error:%v", jsonutils.Marshal(retErr))
			return errors.Wrap(retErr, "send email")
		}
	} else {
		// 构造email发送请求
		gmsg := gomail.NewMessage()
		gmsg.SetHeader("From", models.ConfigMap[api.EMAIL].Content.SenderAddress)
		gmsg.SetHeader("To", args.Receivers.Contact)
		gmsg.SetHeader("Subject", args.Title)
		gmsg.SetBody("text/html", args.EmailMsg.Body)
		dialer.StartTLSPolicy = gomail.MandatoryStartTLS
		if err := dialer.DialAndSend(gmsg); err != nil {
			return errors.Wrap(err, "send email")
		}
	}

	return nil
}

func (emailSender *SEmailSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	errChan := make(chan error, 1)
	go func() {
		dialer := gomail.NewDialer(config.Hostname, config.Hostport, config.Username, config.Password)
		if config.SslGlobal {
			dialer.SSL = true
		} else {
			dialer.SSL = false
			// StartLSConfig
			dialer.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		sender, err := dialer.Dial()
		if err != nil {
			errChan <- err
			return
		}
		sender.Close()
		errChan <- nil
	}()

	ticker := time.Tick(10 * time.Second)
	select {
	case <-ticker:
		return "", errors.Error("timeout")
	case err := <-errChan:
		return "", err
	}
}

func (emailSender *SEmailSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", nil
}

func (emailSender *SEmailSender) IsPersonal() bool {
	return true
}

func (emailSender *SEmailSender) IsRobot() bool {
	return false
}

func (emailSender *SEmailSender) IsValid() bool {
	return len(emailSender.config) > 0
}

func (emailSender *SEmailSender) IsPullType() bool {
	return false
}

func (emailSender *SEmailSender) IsSystemConfigContactType() bool {
	return true
}

func (emailSender *SEmailSender) GetAccessToken(ctx context.Context, key string) error {
	return nil
}

func (emailSender *SEmailSender) sendMessageWithToken(ctx context.Context, uri string, method httputils.THttpMethod, header http.Header, params url.Values, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", models.ConfigMap[api.WORKWX].Content.AccessToken)
	return sendRequest(ctx, uri, httputils.POST, nil, params, jsonutils.Marshal(body))
}

func (emailSender *SEmailSender) RegisterConfig(config models.SConfig) {
	models.ConfigMap[config.Type] = config
}

func init() {
	models.Register(&SEmailSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}

/*
func SendEmail(conf *api.SEmailConfig, msg *api.SEmailMessage) error {
	dialer := gomail.NewDialer(conf.Hostname, conf.Hostport, conf.Username, conf.Password)

	if conf.SslGlobal {
		dialer.SSL = true
	} else {
		dialer.SSL = false
		dialer.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	sender, err := dialer.Dial()
	if err != nil {
		return errors.Wrap(err, "dialer.Dial")
	}

	retErr := errorMap{}

	destMap := make(map[string]int)
	for _, tos := range [][]string{
		msg.To,
		msg.Cc,
		msg.Bcc,
	} {
		for _, to := range tos {
			to = strings.ToLower(to)
			if _, ok := destMap[to]; !ok {
				destMap[to] = 1
			}
		}
	}

	for to := range destMap {
		log.Debugf("send to %s %s", to, msg.Subject)
		gmsg := gomail.NewMessage()
		gmsg.SetHeader("From", conf.SenderAddress)
		gmsg.SetHeader("To", to)

		gmsg.SetHeader("Subject", msg.Subject)
		gmsg.SetBody("text/html", msg.Body)

		for i := range msg.Attachments {
			attach := msg.Attachments[i]
			gmsg.Attach(attach.Filename,
				gomail.SetCopyFunc(func(w io.Writer) error {
					mime := attach.Mime
					if len(mime) == 0 {
						mime = "application/octet-stream"
					}
					_, err := w.Write([]byte("Content-Type: " + attach.Mime))
					return errors.Wrap(err, "WriteMime")
				}),
				gomail.SetCopyFunc(func(w io.Writer) error {
					contBytes, err := base64.StdEncoding.DecodeString(attach.Base64Content)
					if err != nil {
						return errors.Wrap(err, "base64.StdEncoding.DecodeString")
					}
					_, err = w.Write(contBytes)
					return errors.Wrap(err, "WriteContent")
				}),
			)
		}

		errs := make([]error, 0)
		for tryTime := 3; tryTime > 0; tryTime-- {
			err = gomail.Send(sender, gmsg)
			if err != nil {
				errs = append(errs, err)
				time.Sleep(time.Second * 10)
				continue
			}
			errs = errs[0:0]
			break
		}

		if len(errs) > 0 {
			retErr[to] = errors.NewAggregate(errs)
		}
	}
	if len(retErr) > 0 {
		return retErr
	}
	return nil
}
*/
