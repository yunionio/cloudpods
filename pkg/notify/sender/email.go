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
	"crypto/tls"
	"encoding/base64"
	"io"
	"time"

	"gopkg.in/mail.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
)

type errorMap map[string]error

func (em errorMap) Error() string {
	msg := make(map[string]string)
	for k, e := range em {
		msg[k] = e.Error()
	}
	return jsonutils.Marshal(msg).String()
}

func SendEmail(conf *api.SEmailConfig, msg *api.SEmailMessage) error {
	dialer := mail.NewDialer(conf.Hostname, conf.Hostport, conf.Username, conf.Password)

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

	for _, to := range msg.To {
		gmsg := mail.NewMessage()
		gmsg.SetHeader("From", conf.SenderAddress)
		gmsg.SetHeader("To", to)
		gmsg.SetHeader("Subject", msg.Subject)
		gmsg.SetBody("text/html", msg.Body)

		for i := range msg.Attachments {
			attach := msg.Attachments[i]
			gmsg.Attach(attach.Filename,
				mail.SetCopyFunc(func(w io.Writer) error {
					mime := attach.Mime
					if len(mime) == 0 {
						mime = "application/octet-stream"
					}
					_, err := w.Write([]byte("Content-Type: " + attach.Mime))
					return errors.Wrap(err, "WriteMime")
				}),
				mail.SetCopyFunc(func(w io.Writer) error {
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
			err = mail.Send(sender, gmsg)
			log.Debugf("send email ...")
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
