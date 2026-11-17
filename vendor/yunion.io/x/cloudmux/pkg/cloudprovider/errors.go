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

package cloudprovider

import (
	"strings"

	"yunion.io/x/pkg/errors"
)

const (
	ErrMissingParameter    = errors.Error("MissingParameterError")
	ErrInputParameter      = errors.Error("InputParameterError")
	ErrInvalidProvider     = errors.Error("InvalidProvider")
	ErrNoBalancePermission = errors.Error("NoBalancePermission")
	ErrForbidden           = errors.Error("ForbiddenError")
	ErrTooLarge            = errors.Error("TooLargeEntity")
	ErrUnsupportedProtocol = errors.Error("UnsupportedProtocol")
	ErrInvalidAccessKey    = errors.Error("InvalidAccessKey")
	ErrUnauthorized        = errors.Error("UnauthorizedError")
	ErrNoPermission        = errors.Error("NoPermission")
	ErrNoSuchProvder       = errors.Error("no such provider")
	ErrTooManyRequests     = errors.Error("TooManyRequests")
	ErrInvalidSku          = errors.Error("InvalidSku")

	ErrNotFound        = errors.ErrNotFound
	ErrDuplicateId     = errors.ErrDuplicateId
	ErrInvalidStatus   = errors.ErrInvalidStatus
	ErrTimeout         = errors.ErrTimeout
	ErrNotImplemented  = errors.ErrNotImplemented
	ErrNotSupported    = errors.ErrNotSupported
	ErrAccountReadOnly = errors.ErrAccountReadOnly

	ErrUnknown = errors.Error("UnknownError")
)

// IsBucketNotFoundError reports whether err indicates the object-store bucket is gone.
func IsBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Cause(err) == ErrNotFound {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nosuchbucket") ||
		strings.Contains(msg, "no such bucket") ||
		strings.Contains(msg, "the specified bucket does not exist") ||
		strings.Contains(msg, "bucket does not exist") ||
		strings.Contains(msg, "bucket not found") ||
		strings.Contains(msg, "notexistbucket")
}

// ConvertBucketError maps remote bucket-missing errors to ErrNotFound.
func ConvertBucketError(err error) error {
	if err == nil {
		return nil
	}
	if IsBucketNotFoundError(err) {
		return errors.Wrap(ErrNotFound, err.Error())
	}
	return err
}
