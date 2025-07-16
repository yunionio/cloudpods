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

package ssl_certificate

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/eggsampler/acme/v3"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var (
	SSLCertificateCreateWorkerManager *appsrv.SWorkerManager
)

func init() {
	SSLCertificateCreateWorkerManager = appsrv.NewWorkerManager("SSLCertificateCreateWorkerManager", 8, 1024, false)
	taskman.RegisterTaskAndWorker(SSLCertificateCreateTask{}, SSLCertificateCreateWorkerManager)
}

type SSLCertificateCreateTask struct {
	taskman.STask
}

func (self *SSLCertificateCreateTask) taskFailed(ctx context.Context, sc *models.SSSLCertificate, err error) {
	sc.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(sc, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, sc, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func key2pem(certKey *ecdsa.PrivateKey) ([]byte, error) {
	certKeyEnc, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return nil, errors.Wrapf(err, "MarshalECPrivateKey")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certKeyEnc,
	}), nil
}

func (self *SSLCertificateCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sc := obj.(*models.SSSLCertificate)

	zone, err := sc.GetDnsZone()
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "GetDnsZone"))
		return
	}

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "GetProvider"))
		return
	}
	if len(sc.Issuer) == 0 {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "Issuer is required"))
		return
	}

	addr := ""
	switch sc.Issuer {
	case api.SSL_ISSUER_LETSENCRYPT:
		addr = acme.LetsEncryptProduction
	case api.SSL_ISSUER_ZEROSSL:
		addr = acme.ZeroSSLProduction
	}

	client, err := acme.NewClient(addr)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "NewClient"))
		return
	}
	client.PollInterval = 10 * time.Second
	client.PollTimeout = 6 * time.Minute

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "GenerateKey"))
		return
	}

	emails := []string{}
	for _, email := range options.Options.SSLAccounts {
		emails = append(emails, "mailto:"+email)
	}
	account, err := client.NewAccount(privKey, false, true, emails...)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "NewAccount"))
		return
	}

	domainList := strings.Split(sc.Sans, ",")
	var ids []acme.Identifier
	for _, domain := range domainList {
		ids = append(ids, acme.Identifier{Type: "dns", Value: domain})
	}

	order, err := client.NewOrder(account, ids)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "NewOrder"))
		return
	}

	for _, authUrl := range order.Authorizations {
		auth, err := client.FetchAuthorization(account, authUrl)
		if err != nil {
			self.taskFailed(ctx, sc, errors.Wrapf(err, "FetchAuthorization"))
			return
		}
		chal, ok := auth.ChallengeMap[acme.ChallengeTypeDNS01]
		if !ok {
			self.taskFailed(ctx, sc, fmt.Errorf("ChallengeTypeDNS01 not found"))
			return
		}

		txt := acme.EncodeDNS01KeyAuthorization(chal.KeyAuthorization)

		info := strings.Split(auth.Identifier.Value, ".")
		subDomain := strings.Join(info[:len(info)-2], ".")
		subDomain = strings.ReplaceAll(subDomain, "*", "")
		dnsName := "_acme-challenge"
		if len(subDomain) > 0 {
			dnsName = dnsName + "." + subDomain
		}

		log.Debugf("add dns record for %s: label: %s, txt: %s", auth.Identifier.Value, dnsName, txt)

		err = func() error {
			opts := &cloudprovider.DnsRecord{
				DnsName:  dnsName,
				DnsValue: txt,
				DnsType:  cloudprovider.DnsTypeTXT,
				Enabled:  true,
				Ttl:      60,
			}
			recordId, err := iZone.AddDnsRecord(opts)
			if err != nil {
				return errors.Wrapf(err, "AddDnsRecord")
			}

			cloudprovider.Wait(10*time.Second, 3*time.Minute, func() (bool, error) {
				v, err := net.LookupTXT("_acme-challenge." + auth.Identifier.Value)
				log.Debugf("lookup txt for %s: %s, error: %v", "_acme-challenge."+auth.Identifier.Value, v, err)
				if len(v) > 0 {
					return true, nil
				}
				return false, nil
			})

			defer func() {
				record, err := iZone.GetIDnsRecordById(recordId)
				if err != nil {
					logclient.AddActionLogWithStartable(self, sc, logclient.ACT_UPDATE, errors.Wrapf(err, "GetIDnsRecordById"), self.UserCred, false)
					return
				}
				err = record.Delete()
				if err != nil {
					logclient.AddActionLogWithStartable(self, sc, logclient.ACT_UPDATE, errors.Wrapf(err, "Delete"), self.UserCred, false)
					return
				}
			}()

			chal, err = client.UpdateChallenge(account, chal)
			if err != nil {
				return errors.Wrapf(err, "UpdateChallenge")
			}
			return nil
		}()
		if err != nil {
			self.taskFailed(ctx, sc, errors.Wrapf(err, "AddDnsRecord"))
			return
		}
	}

	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "GenerateKey"))
		return
	}

	b, err := key2pem(certKey)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "key2pem"))
		return
	}

	tpl := &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		PublicKeyAlgorithm: x509.ECDSA,
		PublicKey:          certKey.Public(),
		Subject:            pkix.Name{CommonName: domainList[0]},
		DNSNames:           domainList,
	}
	csrDer, err := x509.CreateCertificateRequest(rand.Reader, tpl, certKey)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "CreateCertificateRequest"))
		return
	}
	csr, err := x509.ParseCertificateRequest(csrDer)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "ParseCertificateRequest"))
		return
	}

	order, err = client.FinalizeOrder(account, order, csr)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "FinalizeOrder"))
		return
	}

	certs, err := client.FetchCertificates(account, order.Certificate)
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "FetchCertificates"))
		return
	}

	start, end, country, province, city := time.Time{}, time.Time{}, "", "", ""
	var pemData []string
	for i, c := range certs {
		if i == 0 {
			start = c.NotBefore
			end = c.NotAfter
			if len(c.Subject.Country) > 0 {
				country = c.Subject.Country[0]
			}
			if len(c.Subject.Province) > 0 {
				province = c.Subject.Province[0]
			}
			if len(c.Subject.Locality) > 0 {
				city = c.Subject.Locality[0]
			}
		}
		pemData = append(pemData, strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c.Raw,
		}))))
	}

	_, err = db.Update(sc, func() error {
		sc.Certificate = strings.Join(pemData, "\n")
		sc.PrivateKey = string(b)
		sc.EndDate = end
		sc.StartDate = start
		sc.Country = country
		sc.Province = province
		sc.City = city
		sc.Status = apis.STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		self.taskFailed(ctx, sc, errors.Wrapf(err, "Update"))
		return
	}

	err = func() error {
		provider, err := zone.GetProvider(ctx)
		if err != nil {
			return errors.Wrapf(err, "GetProvider")
		}
		opts := &cloudprovider.SSLCertificateCreateOptions{
			Name:        sc.Name,
			DnsZoneId:   zone.ExternalId,
			Certificate: sc.Certificate,
			PrivateKey:  sc.PrivateKey,
		}
		_, err = provider.CreateISSLCertificate(opts)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported {
				return nil
			}
			return errors.Wrapf(err, "CreateISSLCertificate")
		}
		return nil
	}()
	if err != nil {
		logclient.AddActionLogWithStartable(self, sc, logclient.ACT_CREATE, err, self.UserCred, false)
	}

	self.SetStageComplete(ctx, nil)
}
