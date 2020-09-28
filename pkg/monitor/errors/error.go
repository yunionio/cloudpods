package errors

import (
	"yunion.io/x/onecloud/pkg/httperrors"
)

func NewArgIsEmptyErr(name string) error {
	return httperrors.NewInputParameterError("parameter %s is empty", name)
}
