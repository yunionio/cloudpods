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

package handlers

import (
	"context"
	"net/http"
	"runtime/debug"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func generalError(ctx context.Context, statusCode int, errCode string, msg string) s3cli.ErrorResponse {
	o := fetchObjectRequest(ctx)
	resp := s3cli.ErrorResponse{}
	resp.StatusCode = statusCode
	resp.Code = errCode
	resp.Message = msg
	resp.BucketName = o.Bucket
	resp.Key = o.Key
	resp.HostID = appctx.AppContextHostId(ctx)
	resp.RequestID = appctx.AppContextRequestId(ctx)
	return resp
}

func BadRequest(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 400, "Bad Request", msg)
}

func Unauthenticated(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 401, "Unauthenticated", msg)
}

func Unauthorized(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 403, "Unauthorized", msg)
}

func Forbidden(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 403, "Forbidden", msg)
}

func NotFound(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 404, "Not Found", msg)
}

func NotSupported(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 404, "Not Supported", msg)
}

func NotImplemented(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 406, "Not Implemented", msg)
}

func InvalidStatus(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 406, "Invalid Status", msg)
}

func Conflict(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 409, "Conflict", msg)
}

func ServerTimeout(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 504, "Server Timeout", msg)
}

func ServerError(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 500, "Internal Server Error", msg)
}

func OutOfRangeError(ctx context.Context, msg string) s3cli.ErrorResponse {
	return generalError(ctx, 416, "Range Not Satisfiable", msg)
}

func SendGeneralError(ctx context.Context, w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case s3cli.ErrorResponse:
		SendError(w, e)
	case *s3cli.ErrorResponse:
		SendError(w, *e)
	default:
		var eresp s3cli.ErrorResponse
		cause := errors.Cause(err)
		switch cause {
		case httperrors.ErrUnauthenticated:
			eresp = Unauthenticated(ctx, err.Error())
		case httperrors.ErrUnauthorized:
			eresp = Unauthorized(ctx, err.Error())
		case httperrors.ErrNotFound:
			eresp = NotFound(ctx, err.Error())
		case httperrors.ErrNotSupported:
			eresp = NotSupported(ctx, err.Error())
		case httperrors.ErrNotImplemented:
			eresp = NotImplemented(ctx, err.Error())
		case httperrors.ErrDuplicateId:
			eresp = Conflict(ctx, err.Error())
		case httperrors.ErrTimeout:
			eresp = ServerTimeout(ctx, err.Error())
		case httperrors.ErrInvalidStatus:
			eresp = InvalidStatus(ctx, err.Error())
		case httperrors.ErrBadRequest:
			eresp = BadRequest(ctx, err.Error())
		case httperrors.ErrOutOfRange:
			eresp = OutOfRangeError(ctx, err.Error())
		case httperrors.ErrForbidden:
			eresp = Forbidden(ctx, err.Error())
		default:
			eresp = ServerError(ctx, err.Error())
		}
		SendError(w, eresp)
	}
}

func SendError(w http.ResponseWriter, resp s3cli.ErrorResponse) {
	w.WriteHeader(resp.StatusCode)

	log.Errorf("SendError: %s", resp)
	debug.PrintStack()

	appsrv.SendXml(w, nil, resp)
}
