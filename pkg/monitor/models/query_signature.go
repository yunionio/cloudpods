package models

import (
	"crypto/sha256"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	QUERY_SIGNATURE_KEY = "signature"
)

func digestQuerySignature(data *jsonutils.JSONDict) string {
	data.Remove(QUERY_SIGNATURE_KEY)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data.String())))
}

func ValidateQuerySignature(input jsonutils.JSONObject) error {
	data, ok := input.(*jsonutils.JSONDict)
	if !ok {
		return httperrors.NewInputParameterError("input not json dict")
	}
	signature, err := data.GetString(QUERY_SIGNATURE_KEY)
	if err != nil {
		if errors.Cause(err) == jsonutils.ErrJsonDictKeyNotFound {
			return httperrors.NewNotFoundError("not found signature")
		}
		return errors.Wrap(err, "get signature")
	}
	if signature != digestQuerySignature(data) {

		return httperrors.NewBadRequestError("signature error")
	}
	return nil
}
