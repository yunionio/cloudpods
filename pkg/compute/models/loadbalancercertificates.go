package models

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerCertificateManager struct {
	db.SVirtualResourceBaseManager
}

var LoadbalancerCertificateManager *SLoadbalancerCertificateManager

func init() {
	LoadbalancerCertificateManager = &SLoadbalancerCertificateManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerCertificate{},
			"loadbalancercertificates_tbl",
			"loadbalancercertificate",
			"loadbalancercertificates",
		),
	}
}

// TODO
//
//  - notify users of cert expiration
//  - ca info: self-signed, public ca
type SLoadbalancerCertificate struct {
	db.SVirtualResourceBase

	Certificate string `create:"required" list:"user" update:"user"`
	PrivateKey  string `create:"required" list:"admin" update:"user"`

	// derived attributes
	PublicKeyAlgorithm      string    `create:"optional" list:"user" update:"user"`
	PublicKeyBitLen         int       `create:"optional" list:"user" update:"user"`
	SignatureAlgorithm      string    `create:"optional" list:"user" update:"user"`
	Fingerprint             string    `create:"optional" list:"user" update:"user"`
	NotBefore               time.Time `create:"optional" list:"user" update:"user"`
	NotAfter                time.Time `create:"optional" list:"user" update:"user"`
	CommonName              string    `create:"optional" list:"user" update:"user"`
	SubjectAlternativeNames string    `create:"optional" list:"user" update:"user"`
}

func (man *SLoadbalancerCertificateManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerCertificate{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.PreDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerCertificateManager) validateCertKey(ctx context.Context, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	certV := validators.NewCertificateValidator("certificate")
	pkeyV := validators.NewPrivateKeyValidator("private_key")
	keyV := map[string]validators.IValidator{
		"certificate": certV,
		"private_key": pkeyV,
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	cert := certV.Certificates[0]
	var certPubKeyAlgo string
	{
		// x509.PublicKeyAlgorithm.String() is only available since go1.10
		switch cert.PublicKeyAlgorithm {
		case x509.RSA:
			certPubKeyAlgo = LB_TLS_CERT_PUBKEY_ALGO_RSA
		case x509.ECDSA:
			certPubKeyAlgo = LB_TLS_CERT_PUBKEY_ALGO_ECDSA
		default:
			certPubKeyAlgo = fmt.Sprintf("algo %#v", cert.PublicKeyAlgorithm)
		}
		if !LB_TLS_CERT_PUBKEY_ALGOS.Has(certPubKeyAlgo) {
			return nil, httperrors.NewInputParameterError("invalid cert pubkey algorithm: %s, want %s",
				certPubKeyAlgo, LB_TLS_CERT_PUBKEY_ALGOS.String())
		}
	}
	err := pkeyV.MatchCertificate(cert)
	if err != nil {
		return nil, err
	}
	// NOTE subject alternative names also includes email, url, ip addresses,
	// but we ignore them here.
	//
	// NOTE we use white space to separate names
	data.Set("common_name", jsonutils.NewString(cert.Subject.CommonName))
	data.Set("subject_alternative_names", jsonutils.NewString(strings.Join(cert.DNSNames, " ")))

	data.Set("not_before", jsonutils.NewTimeString(cert.NotBefore))
	data.Set("not_after", jsonutils.NewTimeString(cert.NotAfter))
	data.Set("public_key_algorithm", jsonutils.NewString(certPubKeyAlgo))
	data.Set("public_key_bit_len", jsonutils.NewInt(int64(certV.PublicKeyBitLen())))
	data.Set("signature_algorithm", jsonutils.NewString(cert.SignatureAlgorithm.String()))
	data.Set("fingerprint", jsonutils.NewString(LB_TLS_CERT_FINGERPRINT_ALGO_SHA256+":"+certV.FingerprintSha256String()))
	return data, nil
}

func (man *SLoadbalancerCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := man.validateCertKey(ctx, data)
	if err != nil {
		return nil, err
	}
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (man *SLoadbalancerCertificateManager) InitializeData() error {
	// initialize newly added null certificate fingerprint column
	q := man.Query().IsNull("fingerprint")
	lbcerts := []SLoadbalancerCertificate{}
	if err := q.All(&lbcerts); err != nil {
		return err
	}
	for i := range lbcerts {
		lbcert := &lbcerts[i]
		fp := lbcert.Fingerprint
		if fp != "" {
			continue
		}
		if lbcert.Certificate == "" {
			continue
		}
		{
			p, _ := pem.Decode([]byte(lbcert.Certificate))
			c, err := x509.ParseCertificate(p.Bytes)
			if err != nil {
				log.Errorf("parsing certificate %s(%s): %s", lbcert.Name, lbcert.Id, err)
				continue
			}
			d := sha256.Sum256(c.Raw)
			fp = LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 + ":" + hex.EncodeToString(d[:])
		}
		_, err := man.TableSpec().Update(lbcert, func() error {
			lbcert.Fingerprint = fp
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (lbcert *SLoadbalancerCertificate) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbcert *SLoadbalancerCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !data.Contains("certificate") {
		data.Set("certificate", jsonutils.NewString(lbcert.Certificate))
	}
	if !data.Contains("private_key") {
		data.Set("private_key", jsonutils.NewString(lbcert.PrivateKey))
	}
	data, err := LoadbalancerCertificateManager.validateCertKey(ctx, data)
	if err != nil {
		return nil, err
	}
	return lbcert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbcert *SLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerListenerManager,
	}
	lbcertId := lbcert.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n := t.Query().
			Equals("certificate_id", lbcertId).
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			Count()
		if n > 0 {
			return fmt.Errorf("certificate %s is still referred to by %d %s",
				lbcertId, n, man.KeywordPlural())
		}
	}
	return nil
}

func (lbcert *SLoadbalancerCertificate) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbcert.DoPendingDelete(ctx, userCred)
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}
