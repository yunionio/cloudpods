package models

import "yunion.io/x/onecloud/pkg/httperrors"

type IGetVpc interface {
	GetName() string
	GetVpc() (*SVpc, error)
}

func ValidateAssociateEip(obj IGetVpc) error {
	vpc, err := obj.GetVpc()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if vpc != nil {
		if !vpc.IsSupportAssociateEip() {
			return httperrors.NewNotSupportedError("resource %s in vpc %s external access mode %s is not support accociate eip", obj.GetName(), vpc.GetName(), vpc.ExternalAccessMode)
		}
	}

	return nil
}
