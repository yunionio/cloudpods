// Copyright 2019 Huawei Technologies Co.,Ltd.
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use
// this file except in compliance with the License.  You may obtain a copy of the
// License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations under the License.

//nolint:structcheck, unused
package obs

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// IReadCloser defines interface with function: setReadCloser
type IReadCloser interface {
	setReadCloser(body io.ReadCloser)
}

func (output *GetObjectOutput) setReadCloser(body io.ReadCloser) {
	output.Body = body
}

func setHeaders(headers map[string][]string, header string, headerValue []string, isObs bool) {
	if isObs {
		header = HEADER_PREFIX_OBS + header
		headers[header] = headerValue
	} else {
		header = HEADER_PREFIX + header
		headers[header] = headerValue
	}
}

func setHeadersNext(headers map[string][]string, header string, headerNext string, headerValue []string, isObs bool) {
	if isObs {
		headers[header] = headerValue
	} else {
		headers[headerNext] = headerValue
	}
}

// IBaseModel defines interface for base response model
type IBaseModel interface {
	setStatusCode(statusCode int)

	setRequestID(requestID string)

	setResponseHeaders(responseHeaders map[string][]string)
}

// ISerializable defines interface with function: trans
type ISerializable interface {
	trans(isObs bool) (map[string]string, map[string][]string, interface{}, error)
}

// DefaultSerializable defines default serializable struct
type DefaultSerializable struct {
	params  map[string]string
	headers map[string][]string
	data    interface{}
}

func (input DefaultSerializable) trans(isObs bool) (map[string]string, map[string][]string, interface{}, error) {
	return input.params, input.headers, input.data, nil
}

var defaultSerializable = &DefaultSerializable{}

func newSubResourceSerial(subResource SubResourceType) *DefaultSerializable {
	return &DefaultSerializable{map[string]string{string(subResource): ""}, nil, nil}
}

func trans(subResource SubResourceType, input interface{}) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(subResource): ""}
	data, err = ConvertRequestToIoReader(input)
	return
}

func (baseModel *BaseModel) setStatusCode(statusCode int) {
	baseModel.StatusCode = statusCode
}

func (baseModel *BaseModel) setRequestID(requestID string) {
	baseModel.RequestId = requestID
}

func (baseModel *BaseModel) setResponseHeaders(responseHeaders map[string][]string) {
	baseModel.ResponseHeaders = responseHeaders
}

func (input ListBucketsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string)
	if input.QueryLocation && !isObs {
		setHeaders(headers, HEADER_LOCATION_AMZ, []string{"true"}, isObs)
	}
	if input.BucketType != "" {
		setHeaders(headers, HEADER_BUCKET_TYPE, []string{string(input.BucketType)}, true)
	}
	return
}

func (input CreateBucketInput) prepareGrantHeaders(headers map[string][]string, isObs bool) {
	if grantReadID := input.GrantReadId; grantReadID != "" {
		setHeaders(headers, HEADER_GRANT_READ_OBS, []string{grantReadID}, isObs)
	}
	if grantWriteID := input.GrantWriteId; grantWriteID != "" {
		setHeaders(headers, HEADER_GRANT_WRITE_OBS, []string{grantWriteID}, isObs)
	}
	if grantReadAcpID := input.GrantReadAcpId; grantReadAcpID != "" {
		setHeaders(headers, HEADER_GRANT_READ_ACP_OBS, []string{grantReadAcpID}, isObs)
	}
	if grantWriteAcpID := input.GrantWriteAcpId; grantWriteAcpID != "" {
		setHeaders(headers, HEADER_GRANT_WRITE_ACP_OBS, []string{grantWriteAcpID}, isObs)
	}
	if grantFullControlID := input.GrantFullControlId; grantFullControlID != "" {
		setHeaders(headers, HEADER_GRANT_FULL_CONTROL_OBS, []string{grantFullControlID}, isObs)
	}
	if grantReadDeliveredID := input.GrantReadDeliveredId; grantReadDeliveredID != "" {
		setHeaders(headers, HEADER_GRANT_READ_DELIVERED_OBS, []string{grantReadDeliveredID}, true)
	}
	if grantFullControlDeliveredID := input.GrantFullControlDeliveredId; grantFullControlDeliveredID != "" {
		setHeaders(headers, HEADER_GRANT_FULL_CONTROL_DELIVERED_OBS, []string{grantFullControlDeliveredID}, true)
	}
}

func (input CreateBucketInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string)
	if acl := string(input.ACL); acl != "" {
		setHeaders(headers, HEADER_ACL, []string{acl}, isObs)
	}
	if storageClass := string(input.StorageClass); storageClass != "" {
		if !isObs {
			if storageClass == string(StorageClassWarm) {
				storageClass = string(storageClassStandardIA)
			} else if storageClass == string(StorageClassCold) {
				storageClass = string(storageClassGlacier)
			}
		}
		setHeadersNext(headers, HEADER_STORAGE_CLASS_OBS, HEADER_STORAGE_CLASS, []string{storageClass}, isObs)
	}
	if epid := input.Epid; epid != "" {
		setHeaders(headers, HEADER_EPID_HEADERS, []string{epid}, isObs)
	}
	if availableZone := input.AvailableZone; availableZone != "" {
		setHeaders(headers, HEADER_AZ_REDUNDANCY, []string{availableZone}, isObs)
	}

	input.prepareGrantHeaders(headers, isObs)
	if input.IsFSFileInterface {
		setHeaders(headers, headerFSFileInterface, []string{"Enabled"}, true)
	}

	if location := strings.TrimSpace(input.Location); location != "" {
		input.Location = location

		xml := make([]string, 0, 3)
		xml = append(xml, "<CreateBucketConfiguration>")
		if isObs {
			xml = append(xml, fmt.Sprintf("<Location>%s</Location>", input.Location))
		} else {
			xml = append(xml, fmt.Sprintf("<LocationConstraint>%s</LocationConstraint>", input.Location))
		}
		xml = append(xml, "</CreateBucketConfiguration>")

		data = strings.Join(xml, "")
	}
	return
}

func (input SetBucketStoragePolicyInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	xml := make([]string, 0, 1)
	if !isObs {
		storageClass := "STANDARD"
		if input.StorageClass == StorageClassWarm {
			storageClass = string(storageClassStandardIA)
		} else if input.StorageClass == StorageClassCold {
			storageClass = string(storageClassGlacier)
		}
		params = map[string]string{string(SubResourceStoragePolicy): ""}
		xml = append(xml, fmt.Sprintf("<StoragePolicy><DefaultStorageClass>%s</DefaultStorageClass></StoragePolicy>", storageClass))
	} else {
		if input.StorageClass != StorageClassWarm && input.StorageClass != StorageClassCold {
			input.StorageClass = StorageClassStandard
		}
		params = map[string]string{string(SubResourceStorageClass): ""}
		xml = append(xml, fmt.Sprintf("<StorageClass>%s</StorageClass>", input.StorageClass))
	}
	data = strings.Join(xml, "")
	return
}

func (input ListObjsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = make(map[string]string)
	if input.Prefix != "" {
		params["prefix"] = input.Prefix
	}
	if input.Delimiter != "" {
		params["delimiter"] = input.Delimiter
	}
	if input.MaxKeys > 0 {
		params["max-keys"] = IntToString(input.MaxKeys)
	}
	if input.EncodingType != "" {
		params["encoding-type"] = input.EncodingType
	}
	headers = make(map[string][]string)
	if origin := strings.TrimSpace(input.Origin); origin != "" {
		headers[HEADER_ORIGIN_CAMEL] = []string{origin}
	}
	if requestHeader := strings.TrimSpace(input.RequestHeader); requestHeader != "" {
		headers[HEADER_ACCESS_CONTROL_REQUEST_HEADER_CAMEL] = []string{requestHeader}
	}
	return
}

func (input ListObjectsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.ListObjsInput.trans(isObs)
	if err != nil {
		return
	}
	if input.Marker != "" {
		params["marker"] = input.Marker
	}
	return
}

func (input ListVersionsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.ListObjsInput.trans(isObs)
	if err != nil {
		return
	}
	params[string(SubResourceVersions)] = ""
	if input.KeyMarker != "" {
		params["key-marker"] = input.KeyMarker
	}
	if input.VersionIdMarker != "" {
		params["version-id-marker"] = input.VersionIdMarker
	}
	return
}

func (input ListMultipartUploadsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceUploads): ""}
	if input.Prefix != "" {
		params["prefix"] = input.Prefix
	}
	if input.Delimiter != "" {
		params["delimiter"] = input.Delimiter
	}
	if input.MaxUploads > 0 {
		params["max-uploads"] = IntToString(input.MaxUploads)
	}
	if input.KeyMarker != "" {
		params["key-marker"] = input.KeyMarker
	}
	if input.UploadIdMarker != "" {
		params["upload-id-marker"] = input.UploadIdMarker
	}
	if input.EncodingType != "" {
		params["encoding-type"] = input.EncodingType
	}
	return
}

func (input SetBucketQuotaInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	return trans(SubResourceQuota, input)
}

func (input SetBucketAclInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceAcl): ""}
	headers = make(map[string][]string)

	if acl := string(input.ACL); acl != "" {
		setHeaders(headers, HEADER_ACL, []string{acl}, isObs)
	} else {
		data, _ = convertBucketACLToXML(input.AccessControlPolicy, false, isObs)
	}
	return
}

func (input SetBucketPolicyInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourcePolicy): ""}
	data = strings.NewReader(input.Policy)
	return
}

func (input SetBucketCorsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceCors): ""}
	data, md5, err := ConvertRequestToIoReaderV2(input)
	if err != nil {
		return
	}
	headers = map[string][]string{HEADER_MD5_CAMEL: {md5}}
	return
}

func (input SetBucketVersioningInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	return trans(SubResourceVersioning, input)
}

func (input SetBucketWebsiteConfigurationInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceWebsite): ""}
	data, _ = ConvertWebsiteConfigurationToXml(input.BucketWebsiteConfiguration, false)
	return
}

func (input GetBucketMetadataInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string)
	if origin := strings.TrimSpace(input.Origin); origin != "" {
		headers[HEADER_ORIGIN_CAMEL] = []string{origin}
	}
	if requestHeader := strings.TrimSpace(input.RequestHeader); requestHeader != "" {
		headers[HEADER_ACCESS_CONTROL_REQUEST_HEADER_CAMEL] = []string{requestHeader}
	}
	return
}

func (input SetBucketLoggingConfigurationInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceLogging): ""}
	data, _ = ConvertLoggingStatusToXml(input.BucketLoggingStatus, false, isObs)
	return
}

func (input SetBucketLifecycleConfigurationInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceLifecycle): ""}
	data, md5 := ConvertLifecyleConfigurationToXml(input.BucketLifecyleConfiguration, true, isObs)
	headers = map[string][]string{HEADER_MD5_CAMEL: {md5}}
	return
}

func (input SetBucketEncryptionInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceEncryption): ""}
	data, _ = ConvertEncryptionConfigurationToXml(input.BucketEncryptionConfiguration, false, isObs)
	return
}

func (input SetBucketTaggingInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceTagging): ""}
	data, md5, err := ConvertRequestToIoReaderV2(input)
	if err != nil {
		return
	}
	headers = map[string][]string{HEADER_MD5_CAMEL: {md5}}
	return
}

func (input SetBucketNotificationInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceNotification): ""}
	data, _ = ConvertNotificationToXml(input.BucketNotification, false, isObs)
	return
}

func (input DeleteObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = make(map[string]string)
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	return
}

func (input DeleteObjectsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceDelete): ""}
	if strings.ToLower(input.EncodingType) == "url" {
		for index, object := range input.Objects {
			input.Objects[index].Key = url.QueryEscape(object.Key)
		}
	}
	data, md5 := convertDeleteObjectsToXML(input)
	headers = map[string][]string{HEADER_MD5_CAMEL: {md5}}
	return
}

func (input SetObjectAclInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceAcl): ""}
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	headers = make(map[string][]string)
	if acl := string(input.ACL); acl != "" {
		setHeaders(headers, HEADER_ACL, []string{acl}, isObs)
	} else {
		data, _ = ConvertAclToXml(input.AccessControlPolicy, false, isObs)
	}
	return
}

func (input GetObjectAclInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceAcl): ""}
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	return
}

func (input RestoreObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{string(SubResourceRestore): ""}
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	if !isObs {
		data, err = ConvertRequestToIoReader(input)
	} else {
		data = ConverntObsRestoreToXml(input)
	}
	return
}

// GetEncryption gets the Encryption field value from SseKmsHeader
func (header SseKmsHeader) GetEncryption() string {
	if header.Encryption != "" {
		return header.Encryption
	}
	if !header.isObs {
		return DEFAULT_SSE_KMS_ENCRYPTION
	}
	return DEFAULT_SSE_KMS_ENCRYPTION_OBS
}

// GetKey gets the Key field value from SseKmsHeader
func (header SseKmsHeader) GetKey() string {
	return header.Key
}

// GetEncryption gets the Encryption field value from SseCHeader
func (header SseCHeader) GetEncryption() string {
	if header.Encryption != "" {
		return header.Encryption
	}
	return DEFAULT_SSE_C_ENCRYPTION
}

// GetKey gets the Key field value from SseCHeader
func (header SseCHeader) GetKey() string {
	return header.Key
}

// GetKeyMD5 gets the KeyMD5 field value from SseCHeader
func (header SseCHeader) GetKeyMD5() string {
	if header.KeyMD5 != "" {
		return header.KeyMD5
	}

	if ret, err := Base64Decode(header.GetKey()); err == nil {
		return Base64Md5(ret)
	}
	return ""
}

func setSseHeader(headers map[string][]string, sseHeader ISseHeader, sseCOnly bool, isObs bool) {
	if sseHeader != nil {
		if sseCHeader, ok := sseHeader.(SseCHeader); ok {
			setHeaders(headers, HEADER_SSEC_ENCRYPTION, []string{sseCHeader.GetEncryption()}, isObs)
			setHeaders(headers, HEADER_SSEC_KEY, []string{sseCHeader.GetKey()}, isObs)
			setHeaders(headers, HEADER_SSEC_KEY_MD5, []string{sseCHeader.GetKeyMD5()}, isObs)
		} else if sseKmsHeader, ok := sseHeader.(SseKmsHeader); !sseCOnly && ok {
			sseKmsHeader.isObs = isObs
			setHeaders(headers, HEADER_SSEKMS_ENCRYPTION, []string{sseKmsHeader.GetEncryption()}, isObs)
			if sseKmsHeader.GetKey() != "" {
				setHeadersNext(headers, HEADER_SSEKMS_KEY_OBS, HEADER_SSEKMS_KEY_AMZ, []string{sseKmsHeader.GetKey()}, isObs)
			}
		}
	}
}

func (input GetObjectMetadataInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = make(map[string]string)
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	headers = make(map[string][]string)

	if input.Origin != "" {
		headers[HEADER_ORIGIN_CAMEL] = []string{input.Origin}
	}

	if input.RequestHeader != "" {
		headers[HEADER_ACCESS_CONTROL_REQUEST_HEADER_CAMEL] = []string{input.RequestHeader}
	}
	setSseHeader(headers, input.SseHeader, true, isObs)
	return
}

func (input SetObjectMetadataInput) prepareContentHeaders(headers map[string][]string) {
	if input.ContentDisposition != "" {
		headers[HEADER_CONTENT_DISPOSITION_CAMEL] = []string{input.ContentDisposition}
	}
	if input.ContentEncoding != "" {
		headers[HEADER_CONTENT_ENCODING_CAMEL] = []string{input.ContentEncoding}
	}
	if input.ContentLanguage != "" {
		headers[HEADER_CONTENT_LANGUAGE_CAMEL] = []string{input.ContentLanguage}
	}

	if input.ContentType != "" {
		headers[HEADER_CONTENT_TYPE_CAML] = []string{input.ContentType}
	}
}

func (input SetObjectMetadataInput) prepareStorageClass(headers map[string][]string, isObs bool) {
	if storageClass := string(input.StorageClass); storageClass != "" {
		if !isObs {
			if storageClass == string(StorageClassWarm) {
				storageClass = string(storageClassStandardIA)
			} else if storageClass == string(StorageClassCold) {
				storageClass = string(storageClassGlacier)
			}
		}
		setHeaders(headers, HEADER_STORAGE_CLASS2, []string{storageClass}, isObs)
	}
}

func (input SetObjectMetadataInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = make(map[string]string)
	params = map[string]string{string(SubResourceMetadata): ""}
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	headers = make(map[string][]string)

	if directive := string(input.MetadataDirective); directive != "" {
		setHeaders(headers, HEADER_METADATA_DIRECTIVE, []string{string(input.MetadataDirective)}, isObs)
	} else {
		setHeaders(headers, HEADER_METADATA_DIRECTIVE, []string{string(ReplaceNew)}, isObs)
	}
	if input.CacheControl != "" {
		headers[HEADER_CACHE_CONTROL_CAMEL] = []string{input.CacheControl}
	}
	input.prepareContentHeaders(headers)
	if input.Expires != "" {
		headers[HEADER_EXPIRES_CAMEL] = []string{input.Expires}
	}
	if input.WebsiteRedirectLocation != "" {
		setHeaders(headers, HEADER_WEBSITE_REDIRECT_LOCATION, []string{input.WebsiteRedirectLocation}, isObs)
	}
	input.prepareStorageClass(headers, isObs)
	if input.Metadata != nil {
		for key, value := range input.Metadata {
			key = strings.TrimSpace(key)
			setHeadersNext(headers, HEADER_PREFIX_META_OBS+key, HEADER_PREFIX_META+key, []string{value}, isObs)
		}
	}
	return
}

func (input GetObjectInput) prepareResponseParams(params map[string]string) {
	if input.ResponseCacheControl != "" {
		params[PARAM_RESPONSE_CACHE_CONTROL] = input.ResponseCacheControl
	}
	if input.ResponseContentDisposition != "" {
		params[PARAM_RESPONSE_CONTENT_DISPOSITION] = input.ResponseContentDisposition
	}
	if input.ResponseContentEncoding != "" {
		params[PARAM_RESPONSE_CONTENT_ENCODING] = input.ResponseContentEncoding
	}
	if input.ResponseContentLanguage != "" {
		params[PARAM_RESPONSE_CONTENT_LANGUAGE] = input.ResponseContentLanguage
	}
	if input.ResponseContentType != "" {
		params[PARAM_RESPONSE_CONTENT_TYPE] = input.ResponseContentType
	}
	if input.ResponseExpires != "" {
		params[PARAM_RESPONSE_EXPIRES] = input.ResponseExpires
	}
}

func (input GetObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.GetObjectMetadataInput.trans(isObs)
	if err != nil {
		return
	}
	input.prepareResponseParams(params)
	if input.ImageProcess != "" {
		params[PARAM_IMAGE_PROCESS] = input.ImageProcess
	}
	if input.RangeStart >= 0 && input.RangeEnd > input.RangeStart {
		headers[HEADER_RANGE] = []string{fmt.Sprintf("bytes=%d-%d", input.RangeStart, input.RangeEnd)}
	}

	if input.IfMatch != "" {
		headers[HEADER_IF_MATCH] = []string{input.IfMatch}
	}
	if input.IfNoneMatch != "" {
		headers[HEADER_IF_NONE_MATCH] = []string{input.IfNoneMatch}
	}
	if !input.IfModifiedSince.IsZero() {
		headers[HEADER_IF_MODIFIED_SINCE] = []string{FormatUtcToRfc1123(input.IfModifiedSince)}
	}
	if !input.IfUnmodifiedSince.IsZero() {
		headers[HEADER_IF_UNMODIFIED_SINCE] = []string{FormatUtcToRfc1123(input.IfUnmodifiedSince)}
	}
	return
}

func (input ObjectOperationInput) prepareGrantHeaders(headers map[string][]string) {
	if GrantReadID := input.GrantReadId; GrantReadID != "" {
		setHeaders(headers, HEADER_GRANT_READ_OBS, []string{GrantReadID}, true)
	}
	if GrantReadAcpID := input.GrantReadAcpId; GrantReadAcpID != "" {
		setHeaders(headers, HEADER_GRANT_READ_ACP_OBS, []string{GrantReadAcpID}, true)
	}
	if GrantWriteAcpID := input.GrantWriteAcpId; GrantWriteAcpID != "" {
		setHeaders(headers, HEADER_GRANT_WRITE_ACP_OBS, []string{GrantWriteAcpID}, true)
	}
	if GrantFullControlID := input.GrantFullControlId; GrantFullControlID != "" {
		setHeaders(headers, HEADER_GRANT_FULL_CONTROL_OBS, []string{GrantFullControlID}, true)
	}
}

func (input ObjectOperationInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string)
	params = make(map[string]string)
	if acl := string(input.ACL); acl != "" {
		setHeaders(headers, HEADER_ACL, []string{acl}, isObs)
	}
	input.prepareGrantHeaders(headers)
	if storageClass := string(input.StorageClass); storageClass != "" {
		if !isObs {
			if storageClass == string(StorageClassWarm) {
				storageClass = string(storageClassStandardIA)
			} else if storageClass == string(StorageClassCold) {
				storageClass = string(storageClassGlacier)
			}
		}
		setHeaders(headers, HEADER_STORAGE_CLASS2, []string{storageClass}, isObs)
	}
	if input.WebsiteRedirectLocation != "" {
		setHeaders(headers, HEADER_WEBSITE_REDIRECT_LOCATION, []string{input.WebsiteRedirectLocation}, isObs)

	}
	setSseHeader(headers, input.SseHeader, false, isObs)
	if input.Expires != 0 {
		setHeaders(headers, HEADER_EXPIRES, []string{Int64ToString(input.Expires)}, true)
	}
	if input.Metadata != nil {
		for key, value := range input.Metadata {
			key = strings.TrimSpace(key)
			setHeadersNext(headers, HEADER_PREFIX_META_OBS+key, HEADER_PREFIX_META+key, []string{value}, isObs)
		}
	}
	return
}

func (input PutObjectBasicInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.ObjectOperationInput.trans(isObs)
	if err != nil {
		return
	}

	if input.ContentMD5 != "" {
		headers[HEADER_MD5_CAMEL] = []string{input.ContentMD5}
	}

	if input.ContentLength > 0 {
		headers[HEADER_CONTENT_LENGTH_CAMEL] = []string{Int64ToString(input.ContentLength)}
	}
	if input.ContentType != "" {
		headers[HEADER_CONTENT_TYPE_CAML] = []string{input.ContentType}
	}

	return
}

func (input PutObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.PutObjectBasicInput.trans(isObs)
	if err != nil {
		return
	}
	if input.Body != nil {
		data = input.Body
	}
	return
}

func (input AppendObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.PutObjectBasicInput.trans(isObs)
	if err != nil {
		return
	}
	params[string(SubResourceAppend)] = ""
	params["position"] = strconv.FormatInt(input.Position, 10)
	if input.Body != nil {
		data = input.Body
	}
	return
}

func (input ModifyObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string)
	params = make(map[string]string)
	params[string(SubResourceModify)] = ""
	params["position"] = strconv.FormatInt(input.Position, 10)
	if input.ContentLength > 0 {
		headers[HEADER_CONTENT_LENGTH_CAMEL] = []string{Int64ToString(input.ContentLength)}
	}

	if input.Body != nil {
		data = input.Body
	}
	return
}

func (input CopyObjectInput) prepareReplaceHeaders(headers map[string][]string) {
	if input.CacheControl != "" {
		headers[HEADER_CACHE_CONTROL] = []string{input.CacheControl}
	}
	if input.ContentDisposition != "" {
		headers[HEADER_CONTENT_DISPOSITION] = []string{input.ContentDisposition}
	}
	if input.ContentEncoding != "" {
		headers[HEADER_CONTENT_ENCODING] = []string{input.ContentEncoding}
	}
	if input.ContentLanguage != "" {
		headers[HEADER_CONTENT_LANGUAGE] = []string{input.ContentLanguage}
	}
	if input.ContentType != "" {
		headers[HEADER_CONTENT_TYPE] = []string{input.ContentType}
	}
	if input.Expires != "" {
		headers[HEADER_EXPIRES] = []string{input.Expires}
	}
}

func (input CopyObjectInput) prepareCopySourceHeaders(headers map[string][]string, isObs bool) {
	if input.CopySourceIfMatch != "" {
		setHeaders(headers, HEADER_COPY_SOURCE_IF_MATCH, []string{input.CopySourceIfMatch}, isObs)
	}
	if input.CopySourceIfNoneMatch != "" {
		setHeaders(headers, HEADER_COPY_SOURCE_IF_NONE_MATCH, []string{input.CopySourceIfNoneMatch}, isObs)
	}
	if !input.CopySourceIfModifiedSince.IsZero() {
		setHeaders(headers, HEADER_COPY_SOURCE_IF_MODIFIED_SINCE, []string{FormatUtcToRfc1123(input.CopySourceIfModifiedSince)}, isObs)
	}
	if !input.CopySourceIfUnmodifiedSince.IsZero() {
		setHeaders(headers, HEADER_COPY_SOURCE_IF_UNMODIFIED_SINCE, []string{FormatUtcToRfc1123(input.CopySourceIfUnmodifiedSince)}, isObs)
	}
}

func (input CopyObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.ObjectOperationInput.trans(isObs)
	if err != nil {
		return
	}

	var copySource string
	if input.CopySourceVersionId != "" {
		copySource = fmt.Sprintf("%s/%s?versionId=%s", input.CopySourceBucket, UrlEncode(input.CopySourceKey, false), input.CopySourceVersionId)
	} else {
		copySource = fmt.Sprintf("%s/%s", input.CopySourceBucket, UrlEncode(input.CopySourceKey, false))
	}
	setHeaders(headers, HEADER_COPY_SOURCE, []string{copySource}, isObs)

	if directive := string(input.MetadataDirective); directive != "" {
		setHeaders(headers, HEADER_METADATA_DIRECTIVE, []string{directive}, isObs)
	}

	if input.MetadataDirective == ReplaceMetadata {
		input.prepareReplaceHeaders(headers)
	}

	input.prepareCopySourceHeaders(headers, isObs)
	if input.SourceSseHeader != nil {
		if sseCHeader, ok := input.SourceSseHeader.(SseCHeader); ok {
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_ENCRYPTION, []string{sseCHeader.GetEncryption()}, isObs)
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_KEY, []string{sseCHeader.GetKey()}, isObs)
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_KEY_MD5, []string{sseCHeader.GetKeyMD5()}, isObs)
		}
	}
	if input.SuccessActionRedirect != "" {
		headers[HEADER_SUCCESS_ACTION_REDIRECT] = []string{input.SuccessActionRedirect}
	}
	return
}

func (input AbortMultipartUploadInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{"uploadId": input.UploadId}
	return
}

func (input InitiateMultipartUploadInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params, headers, data, err = input.ObjectOperationInput.trans(isObs)
	if err != nil {
		return
	}
	if input.ContentType != "" {
		headers[HEADER_CONTENT_TYPE_CAML] = []string{input.ContentType}
	}
	params[string(SubResourceUploads)] = ""
	if input.EncodingType != "" {
		params["encoding-type"] = input.EncodingType
	}
	return
}

func (input UploadPartInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{"uploadId": input.UploadId, "partNumber": IntToString(input.PartNumber)}
	headers = make(map[string][]string)
	setSseHeader(headers, input.SseHeader, true, isObs)
	if input.ContentMD5 != "" {
		headers[HEADER_MD5_CAMEL] = []string{input.ContentMD5}
	}
	if input.Body != nil {
		data = input.Body
	}
	return
}

func (input CompleteMultipartUploadInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{"uploadId": input.UploadId}
	if input.EncodingType != "" {
		params["encoding-type"] = input.EncodingType
	}
	data, _ = ConvertCompleteMultipartUploadInputToXml(input, false)
	return
}

func (input ListPartsInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{"uploadId": input.UploadId}
	if input.MaxParts > 0 {
		params["max-parts"] = IntToString(input.MaxParts)
	}
	if input.PartNumberMarker > 0 {
		params["part-number-marker"] = IntToString(input.PartNumberMarker)
	}
	if input.EncodingType != "" {
		params["encoding-type"] = input.EncodingType
	}
	return
}

func (input CopyPartInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = map[string]string{"uploadId": input.UploadId, "partNumber": IntToString(input.PartNumber)}
	headers = make(map[string][]string, 1)
	var copySource string
	if input.CopySourceVersionId != "" {
		copySource = fmt.Sprintf("%s/%s?versionId=%s", input.CopySourceBucket, UrlEncode(input.CopySourceKey, false), input.CopySourceVersionId)
	} else {
		copySource = fmt.Sprintf("%s/%s", input.CopySourceBucket, UrlEncode(input.CopySourceKey, false))
	}
	setHeaders(headers, HEADER_COPY_SOURCE, []string{copySource}, isObs)
	if input.CopySourceRangeStart >= 0 && input.CopySourceRangeEnd > input.CopySourceRangeStart {
		setHeaders(headers, HEADER_COPY_SOURCE_RANGE, []string{fmt.Sprintf("bytes=%d-%d", input.CopySourceRangeStart, input.CopySourceRangeEnd)}, isObs)
	}

	setSseHeader(headers, input.SseHeader, true, isObs)
	if input.SourceSseHeader != nil {
		if sseCHeader, ok := input.SourceSseHeader.(SseCHeader); ok {
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_ENCRYPTION, []string{sseCHeader.GetEncryption()}, isObs)
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_KEY, []string{sseCHeader.GetKey()}, isObs)
			setHeaders(headers, HEADER_SSEC_COPY_SOURCE_KEY_MD5, []string{sseCHeader.GetKeyMD5()}, isObs)
		}

	}
	return
}

func (input HeadObjectInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	params = make(map[string]string)
	if input.VersionId != "" {
		params[PARAM_VERSION_ID] = input.VersionId
	}
	return
}

func (input SetBucketRequestPaymentInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	return trans(SubResourceRequestPayment, input)
}

type partSlice []Part

func (parts partSlice) Len() int {
	return len(parts)
}

func (parts partSlice) Less(i, j int) bool {
	return parts[i].PartNumber < parts[j].PartNumber
}

func (parts partSlice) Swap(i, j int) {
	parts[i], parts[j] = parts[j], parts[i]
}

type readerWrapper struct {
	reader      io.Reader
	mark        int64
	totalCount  int64
	readedCount int64
}

func (rw *readerWrapper) seek(offset int64, whence int) (int64, error) {
	if r, ok := rw.reader.(*strings.Reader); ok {
		return r.Seek(offset, whence)
	} else if r, ok := rw.reader.(*bytes.Reader); ok {
		return r.Seek(offset, whence)
	} else if r, ok := rw.reader.(*os.File); ok {
		return r.Seek(offset, whence)
	}
	return offset, nil
}

func (rw *readerWrapper) Read(p []byte) (n int, err error) {
	if rw.totalCount == 0 {
		return 0, io.EOF
	}
	if rw.totalCount > 0 {
		n, err = rw.reader.Read(p)
		readedOnce := int64(n)
		remainCount := rw.totalCount - rw.readedCount
		if remainCount > readedOnce {
			rw.readedCount += readedOnce
			return n, err
		}
		rw.readedCount += remainCount
		return int(remainCount), io.EOF
	}
	return rw.reader.Read(p)
}

type fileReaderWrapper struct {
	readerWrapper
	filePath string
}

func (input SetBucketFetchPolicyInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	contentType, _ := mimeTypes["json"]
	headers = make(map[string][]string, 2)
	headers[HEADER_CONTENT_TYPE] = []string{contentType}
	setHeaders(headers, headerOefMarker, []string{"yes"}, isObs)
	data, err = convertFetchPolicyToJSON(input)
	return
}

func (input GetBucketFetchPolicyInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string, 1)
	setHeaders(headers, headerOefMarker, []string{"yes"}, isObs)
	return
}

func (input DeleteBucketFetchPolicyInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string, 1)
	setHeaders(headers, headerOefMarker, []string{"yes"}, isObs)
	return
}

func (input SetBucketFetchJobInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	contentType, _ := mimeTypes["json"]
	headers = make(map[string][]string, 2)
	headers[HEADER_CONTENT_TYPE] = []string{contentType}
	setHeaders(headers, headerOefMarker, []string{"yes"}, isObs)
	data, err = convertFetchJobToJSON(input)
	return
}

func (input GetBucketFetchJobInput) trans(isObs bool) (params map[string]string, headers map[string][]string, data interface{}, err error) {
	headers = make(map[string][]string, 1)
	setHeaders(headers, headerOefMarker, []string{"yes"}, isObs)
	return
}
