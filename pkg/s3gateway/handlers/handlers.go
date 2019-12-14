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
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/pkg/s3utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
	"yunion.io/x/onecloud/pkg/s3gateway/options"
)

func InitHandlers(app *appsrv.Application) {
	h := app.AddHandler2("HEAD", "", s3authenticate(headHandler), nil, "head", nil)
	h.SetProcessTimeoutCallback(s3HandlerTimeoutInfo)
	h = app.AddHandler2("GET", "", s3authenticate(readHandler), nil, "get", nil)
	h.SetProcessTimeoutCallback(s3HandlerTimeoutInfo)
	h = app.AddHandler2("PUT", "", s3authenticate(putHandler), nil, "put", nil)
	h.SetProcessTimeoutCallback(s3HandlerTimeoutInfo)
	h = app.AddHandler2("POST", "", s3authenticate(postHandler), nil, "post", nil)
	h.SetProcessTimeoutCallback(s3HandlerTimeoutInfo)
	h = app.AddHandler2("DELETE", "", s3authenticate(deleteHandler), nil, "delete", nil)
	h.SetProcessTimeoutCallback(s3HandlerTimeoutInfo)
}

func s3HandlerTimeoutInfo(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	o, _ := getObjectRequest(r)
	if len(o.Bucket) > 0 && len(o.Key) > 0 {
		if r.Method == http.MethodGet && len(r.URL.RawQuery) == 0 {
			return 2 * time.Hour
		} else if r.Method == http.MethodPut && (len(r.URL.RawQuery) == 0 || strings.Contains(r.URL.RawQuery, "partNumber=")) {
			return 2 * time.Hour
		}
	}
	return time.Duration(0)
}

type SObjectRequest struct {
	VirtualHost bool
	Bucket      string
	Key         string
}

func (o SObjectRequest) Validate() error {
	if len(o.Bucket) == 0 {
		return nil
	}
	err := s3utils.CheckValidBucketNameStrict(o.Bucket)
	if err != nil {
		return err
	}
	if len(o.Key) == 0 {
		return nil
	}
	err = s3utils.CheckValidObjectName(o.Key)
	if err != nil {
		return err
	}
	return nil
}

func getObjectRequest(r *http.Request) (SObjectRequest, error) {
	o := SObjectRequest{}
	if regutils.MatchIP4Addr(r.Host) || r.Host == options.Options.DomainName {
		o.VirtualHost = false
		segs := appsrv.SplitPath(r.URL.Path)
		if len(segs) > 0 {
			o.Bucket = segs[0]
			if len(segs) > 1 {
				o.Key = strings.Join(segs[1:], "/")
				if strings.HasSuffix(r.URL.Path, "/") {
					o.Key += "/"
				}
			}
		}
	} else if strings.HasSuffix(r.Host, "."+options.Options.DomainName) {
		o.VirtualHost = true
		o.Bucket = r.Host[:len(r.Host)-len(options.Options.DomainName)-1]
		segs := appsrv.SplitPath(r.URL.Path)
		o.Key = strings.Join(segs, "/")
		if strings.HasSuffix(r.URL.Path, "/") {
			o.Key += "/"
		}
	} else {
		return o, errors.Error("invalid S3 request")
	}
	var err error
	o.Key, err = url.PathUnescape(o.Key)
	if err != nil {
		return o, errors.Wrap(err, "url.PathUnescape")
	}
	return o, o.Validate()
}

func headHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	o := fetchObjectRequest(ctx)
	userCred := auth.FetchUserCredential(ctx, nil)
	if len(o.Bucket) > 0 && len(o.Key) == 0 {
		// head bucket
		err := headBucket(ctx, userCred, o.Bucket)
		if err != nil {
			SendGeneralError(ctx, w, err)
		} else {
			appsrv.SendHeader(w, nil)
		}
		return
	} else if len(o.Bucket) > 0 && len(o.Key) > 0 {
		// head object
		hdr, err := headObject(ctx, userCred, o.Bucket, o.Key)
		if err != nil {
			SendGeneralError(ctx, w, err)
		} else {
			appsrv.SendHeader(w, hdr)
		}
		return
	} else {
		// do nothing
	}
	SendError(w, NotSupported(ctx, "method not supported"))
}

func readBucket(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, query jsonutils.JSONObject, r *http.Request) (interface{}, http.Header, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	if query.Contains("accelerate") {

	} else if query.Contains("acl") {
		resp, err := bucketAcl(ctx, userCred, bucketName)
		return resp, nil, err
	} else if query.Contains("analytics") {

	} else if query.Contains("cors") {

	} else if query.Contains("encryption") {

	} else if query.Contains("inventory") {

	} else if query.Contains("lifecycle") {

	} else if query.Contains("location") {
		result := s3cli.LocationConstraint(bucket.Location)
		return &result, nil, nil
	} else if query.Contains("publicAccessBlock") {

	} else if query.Contains("logging") {

	} else if query.Contains("metrics") {

	} else if query.Contains("notification") {

	} else if query.Contains("object-lock") {

	} else if query.Contains("policyStatus") {

	} else if query.Contains("versions") {

	} else if query.Contains("policy") {

	} else if query.Contains("replication") {

	} else if query.Contains("requestPayment") {

	} else if query.Contains("tagging") {

	} else if query.Contains("versioning") {
		return &s3cli.VersioningConfiguration{}, nil, nil
	} else if query.Contains("website") {

	} else if query.Contains("uploads") {
		input := s3cli.ListMultipartUploadsInput{}
		err := query.Unmarshal(&input)
		if err != nil {
			return nil, nil, errors.Wrap(err, "query.Unmarshal ListMultipartUploadsInput")
		}
		result, err := listBucketUploads(ctx, userCred, bucketName, &input)
		if err != nil {
			return nil, nil, errors.Wrap(err, "listBucketUploads")
		}
		return result, nil, nil
	} else {
		// list objects in bucket
		input := s3cli.ListObjectInput{}
		err := query.Unmarshal(&input)
		if err != nil {
			return nil, nil, errors.Wrap(err, "query.Unmarshal")
		}
		result, err := bucket.ListObject(ctx, userCred, &input)
		if err != nil {
			return nil, nil, errors.Wrap(err, "bucket.ListObject")
		}
		return result, nil, nil
	}
	return nil, nil, NotImplemented(ctx, "not implemented")
}

func readObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, objKey string, query jsonutils.JSONObject, r *http.Request) (interface{}, http.Header, error) {
	if query.Contains("acl") {
		resp, err := objectAcl(ctx, userCred, bucketName, objKey)
		return resp, nil, err
	} else if query.Contains("legal-hold") {

	} else if query.Contains("retention") {

	} else if query.Contains("tagging") {

	} else if query.Contains("torrent") {

	} else {
		// download object itself, which has been handled
	}
	return nil, nil, NotImplemented(ctx, "not implemented")
}

func getRangeOpt(rangeStr string, sizeBytes int64) (*cloudprovider.SGetObjectRange, error) {
	if len(rangeStr) > 0 {
		rangeOptObj := cloudprovider.ParseRange(rangeStr)
		if rangeOptObj.End == 0 {
			rangeOptObj.End = sizeBytes - 1
		}
		if rangeOptObj.Start >= sizeBytes || rangeOptObj.End >= sizeBytes {
			return nil, httperrors.ErrOutOfRange
		}
		if rangeOptObj.Start > 0 || rangeOptObj.End < sizeBytes-1 {
			return &rangeOptObj, nil
		}
	}
	return nil, nil
}

func downloadObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string, reqHdr http.Header, w http.ResponseWriter) error {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "bucket.GetIBucket")
	}
	obj, err := cloudprovider.GetIObject(iBucket, key)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.GetIObject")
	}
	hdr := cloudprovider.MetaToHttpHeader(cloudprovider.META_HEADER_PREFIX, obj.GetMeta())
	eTag := obj.GetETag()
	if len(eTag) > 0 {
		hdr.Set("ETag", eTag)
	}
	lastModified := obj.GetLastModified()
	if !lastModified.IsZero() {
		hdr.Set("Last-Modified", lastModified.Format(timeutils.RFC2882Format))
	}
	rangeStr := reqHdr.Get(http.CanonicalHeaderKey("range"))
	rangeOpt, err := getRangeOpt(rangeStr, obj.GetSizeBytes())
	if err != nil {
		return errors.Wrap(err, rangeStr)
	}
	stream, err := iBucket.GetObject(ctx, key, rangeOpt)
	if err != nil {
		return errors.Wrap(err, "iBucket.GetObject")
	}
	err = appsrv.SendStream(w, rangeOpt != nil, hdr, stream, obj.GetSizeBytes())
	if err != nil {
		return errors.Wrap(err, "appsrv.SendStream")
	}
	return nil
}

func readHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	o := fetchObjectRequest(ctx)
	userCred := auth.FetchUserCredential(ctx, nil)
	if len(o.Bucket) == 0 {
		// service
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		input := s3cli.ListBucketsInput{}
		err = query.Unmarshal(&input)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
		} else {
			resp, err := listService(ctx, userCred, input)
			if err != nil {
				SendGeneralError(ctx, w, err)
			} else {
				appsrv.SendXml(w, nil, resp)
			}
		}
	} else if len(o.Bucket) > 0 && len(o.Key) == 0 {
		// bucket get
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, respHdr, err := readBucket(ctx, userCred, o.Bucket, query, r)
		if err != nil {
			SendGeneralError(ctx, w, err)
			return
		}
		appsrv.SendXml(w, respHdr, resp)
	} else {
		// object get
		if len(r.URL.RawQuery) == 0 {
			// download object
			err := downloadObject(ctx, userCred, o.Bucket, o.Key, r.Header, w)
			if err != nil {
				SendGeneralError(ctx, w, err)
			}
			return
		}
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, respHdr, err := readObject(ctx, userCred, o.Bucket, o.Key, query, r)
		if err != nil {
			SendGeneralError(ctx, w, err)
			return
		}
		if resp != nil {
			appsrv.SendXml(w, respHdr, resp)
		}
	}
}

func postObject(ctx context.Context, userCred mcclient.TokenCredential, bucket string, key string, query jsonutils.JSONObject, r *http.Request) (interface{}, http.Header, error) {
	if query.Contains("uploads") {
		// initialize multipart upload
		return initMultipartUpload(ctx, userCred, r.Header, bucket, key)
	} else if query.Contains("uploadId") {
		// complete multipart upload
		uploadId, err := query.GetString("uploadId")
		if err != nil || len(uploadId) == 0 {
			return nil, nil, errors.Wrap(httperrors.ErrBadRequest, "uploadId")
		}
		request := s3cli.CompleteMultipartUpload{}
		err = appsrv.FetchXml(r, &request)
		if err != nil {
			return nil, nil, errors.Wrap(httperrors.ErrBadRequest, "FetchXml")
		}
		return completeMultipartUpload(ctx, userCred, r.Header, bucket, key, uploadId, &request)
	} else if query.Contains("select") {
		// select object
		return selectObject(ctx, userCred, r.Header, bucket, key)
	} else {
		// upload object by form POST
	}
	return nil, nil, NotImplemented(ctx, "not implemented")
}

func postHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	o := fetchObjectRequest(ctx)
	userCred := auth.FetchUserCredential(ctx, nil)
	if len(o.Bucket) == 0 {
		// no bucket
		// do nothing
	} else if len(o.Bucket) > 0 && len(o.Key) == 0 {
		// bucket post
		// do nothing
	} else {
		// object post
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, respHdr, err := postObject(ctx, userCred, o.Bucket, o.Key, query, r)
		if err != nil {
			SendGeneralError(ctx, w, err)
			return
		}
		appsrv.SendXml(w, respHdr, resp)
		return
	}
	SendError(w, NotSupported(ctx, "method not supported"))
}

func putBucket(ctx context.Context, userCred mcclient.TokenCredential, bucket string, query jsonutils.JSONObject, r *http.Request) (interface{}, http.Header, error) {
	if query.Contains("accelerate") {

	} else if query.Contains("acl") {

	} else if query.Contains("analytics") {

	} else if query.Contains("cors") {

	} else if query.Contains("encryption") {

	} else if query.Contains("inventory") {

	} else if query.Contains("lifecycle") {

	} else if query.Contains("publicAccessBlock") {

	} else if query.Contains("logging") {

	} else if query.Contains("metrics") {

	} else if query.Contains("notification") {

	} else if query.Contains("object-lock") {

	} else if query.Contains("policy") {

	} else if query.Contains("replication") {

	} else if query.Contains("requestPayment") {

	} else if query.Contains("tagging") {

	} else if query.Contains("versioning") {

	} else if query.Contains("website") {

	} else {
		// create bucket
		return nil, nil, NotSupported(ctx, "Not supported")
	}
	return nil, nil, NotImplemented(ctx, "not implemented")
}

func putObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string, query jsonutils.JSONObject, r *http.Request) (interface{}, http.Header, error) {
	if query.Contains("legal-hold") {

	} else if query.Contains("retention") {

	} else if query.Contains("acl") {

	} else if query.Contains("tagging") {

	} else {
		// upload object
		uploadId, _ := query.GetString("uploadId")
		partNumber, _ := query.Int("partNumber")
		copySource := r.Header.Get(http.CanonicalHeaderKey("x-amz-copy-source"))
		if len(copySource) > 0 {
			return copyObject(ctx, userCred, bucketName, key, copySource, r.Header, uploadId, int(partNumber))
		} else {
			hdr, err := uploadObject(ctx, userCred, bucketName, key, r.Header, r.Body, uploadId, int(partNumber))
			defer r.Body.Close()
			if err != nil {
				return nil, nil, err
			}
			return nil, hdr, nil
		}
	}
	return nil, nil, NotImplemented(ctx, "not implemented")
}

func putHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	o := fetchObjectRequest(ctx)
	userCred := auth.FetchUserCredential(ctx, nil)
	if len(o.Bucket) == 0 {
		// no bucket
	} else if len(o.Bucket) > 0 && len(o.Key) == 0 {
		// bucket put
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, respHdr, err := putBucket(ctx, userCred, o.Bucket, query, r)
		if err != nil {
			SendGeneralError(ctx, w, err)
			return
		}
		appsrv.SendXml(w, respHdr, resp)
		return
	} else {
		// object put
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, respHdr, err := putObject(ctx, userCred, o.Bucket, o.Key, query, r)
		if err != nil {
			SendGeneralError(ctx, w, err)
			return
		}
		appsrv.SendXml(w, respHdr, resp)
		return
	}
	SendError(w, NotSupported(ctx, "method not supported"))
}

func deleteBucket(ctx context.Context, userCred mcclient.TokenCredential, bucket string, query jsonutils.JSONObject) (interface{}, error) {
	if query.Contains("analytics") {

	} else if query.Contains("cors") {

	} else if query.Contains("encryption") {

	} else if query.Contains("inventory") {

	} else if query.Contains("lifecycle") {

	} else if query.Contains("publicAccessBlock") {

	} else if query.Contains("metrics") {

	} else if query.Contains("policy") {

	} else if query.Contains("replication") {

	} else if query.Contains("tagging") {

	} else if query.Contains("website") {

	} else {
		// delete bucket
		err := removeBucket(ctx, userCred, bucket)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return nil, NotImplemented(ctx, "not implemented")
}

func deleteObject(ctx context.Context, userCred mcclient.TokenCredential, bucket string, key string, query jsonutils.JSONObject) (interface{}, error) {
	if query.Contains("tagging") {
		return deleteObjectTags(ctx, userCred, bucket, key)
	} else {
		// delete object
		err := removeObject(ctx, userCred, bucket, key)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	o := fetchObjectRequest(ctx)
	userCred := auth.FetchUserCredential(ctx, nil)
	if len(o.Bucket) == 0 {
		// no bucket
	} else if len(o.Bucket) > 0 && len(o.Key) == 0 {
		// bucket delete
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, err := deleteBucket(ctx, userCred, o.Bucket, query)
		if err != nil {
			SendGeneralError(ctx, w, err)
		} else {
			appsrv.SendXml(w, nil, resp)
		}
		return
	} else {
		// object delete
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		resp, err := deleteObject(ctx, userCred, o.Bucket, o.Key, query)
		if err != nil {
			SendGeneralError(ctx, w, err)
		} else {
			appsrv.SendXml(w, nil, resp)
		}
		return
	}
	SendError(w, NotSupported(ctx, "method not supported"))
}
