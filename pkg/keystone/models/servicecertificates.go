package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SServiceCertificateManager struct {
	db.SStandaloneResourceBaseManager
}

var ServiceCertificateManager *SServiceCertificateManager

func init() {
	ServiceCertificateManager = &SServiceCertificateManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SServiceCertificate{},
			"servicecertificates_tbl",
			"servicecertificate",
			"servicecertificates",
		),
	}
	ServiceCertificateManager.SetVirtualObject(ServiceCertificateManager)
}

type SServiceCertificate struct {
	db.SStandaloneResourceBase
	db.SCertificateBase

	CaCertificate string `create:"optional" list:"admin"`
	CaPrivateKey  string `create:"optional" list:"admin"`
}

func (man *SServiceCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := validators.ValidateCertKey(ctx, data)
	if err != nil {
		return nil, err
	}

	// validate ca cert key
	certV := validators.NewCertificateValidator("ca_certificate")
	pkeyV := validators.NewPrivateKeyValidator("ca_private_key")
	keyV := map[string]validators.IValidator{
		"ca_certificate": certV,
		"ca_private_key": pkeyV,
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}

	input := apis.StandaloneResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (cert *SServiceCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("certificate") || data.Contains("private_key") {
		return nil, httperrors.NewForbiddenError("not allowed update content of certificate")
	}
	if data.Contains("ca_certificate") || data.Contains("ca_private_key") {
		return nil, httperrors.NewForbiddenError("not allowed update content of ca certificate")
	}

	updateData := jsonutils.NewDict()
	if name, err := data.GetString("name"); err == nil {
		updateData.Set("name", jsonutils.NewString(name))
	}

	if desc, err := data.GetString("description"); err == nil {
		updateData.Set("description", jsonutils.NewString(desc))
	}

	input := apis.StandaloneResourceBaseUpdateInput{}
	err := updateData.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = cert.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	updateData.Update(jsonutils.Marshal(input))

	return updateData, nil
}
