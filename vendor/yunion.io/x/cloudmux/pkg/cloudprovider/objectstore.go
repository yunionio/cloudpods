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
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"
)

type TBucketACLType string

const (
	// 50 MB
	MAX_PUT_OBJECT_SIZEBYTES = int64(1024 * 1024 * 50)

	// ACLDefault = TBucketACLType("default")

	ACLPrivate         = TBucketACLType(s3cli.CANNED_ACL_PRIVATE)
	ACLAuthRead        = TBucketACLType(s3cli.CANNED_ACL_AUTH_READ)
	ACLPublicRead      = TBucketACLType(s3cli.CANNED_ACL_PUBLIC_READ)
	ACLPublicReadWrite = TBucketACLType(s3cli.CANNED_ACL_PUBLIC_READ_WRITE)
	ACLUnknown         = TBucketACLType("")

	META_HEADER_CACHE_CONTROL       = "Cache-Control"
	META_HEADER_CONTENT_TYPE        = "Content-Type"
	META_HEADER_CONTENT_DISPOSITION = "Content-Disposition"
	META_HEADER_CONTENT_ENCODING    = "Content-Encoding"
	META_HEADER_CONTENT_LANGUAGE    = "Content-Language"
	META_HEADER_CONTENT_MD5         = "Content-MD5"

	META_HEADER_PREFIX = "X-Yunion-Meta-"
)

type SBucketStats struct {
	SizeBytes   int64
	ObjectCount int
}

func (s SBucketStats) Equals(s2 SBucketStats) bool {
	if s.SizeBytes == s2.SizeBytes && s.ObjectCount == s2.ObjectCount {
		return true
	} else {
		return false
	}
}

type SBucketAccessUrl struct {
	Url         string
	Description string
	Primary     bool
}

type SBucketWebsiteRoutingRule struct {
	ConditionErrorCode string
	ConditionPrefix    string

	RedirectProtocol         string
	RedirectReplaceKey       string
	RedirectReplaceKeyPrefix string
}

type SBucketWebsiteConf struct {
	// 主页
	Index string
	// 错误时返回的文档
	ErrorDocument string
	// http或https
	Protocol string

	Rules []SBucketWebsiteRoutingRule
	// 网站访问url,一般由bucketid，region等组成
	Url string
}

type SBucketCORSRule struct {
	AllowedMethods []string
	// 允许的源站，可以设为*
	AllowedOrigins []string
	AllowedHeaders []string
	MaxAgeSeconds  int
	ExposeHeaders  []string
	// 规则区别标识
	Id string
}

type SBucketRefererConf struct {
	// 域名列表
	DomainList []string
	// 域名列表
	// enmu: Black-List, White-List
	RefererType string
	// 是否允许空referer 访问
	AllowEmptyRefer bool

	Enabled bool
}

type SBucketPolicyStatement struct {
	// 授权的目标主体
	Principal map[string][]string `json:"Principal,omitempty"`
	// 授权的行为
	Action []string `json:"Action,omitempty"`
	// Allow|Deny
	Effect string `json:"Effect,omitempty"`
	// 被授权的资源
	Resource []string `json:"Resource,omitempty"`
	// 触发授权的条件
	Condition map[string]map[string]interface{} `json:"Condition,omitempty"`

	// 解析字段，主账号id:子账号id
	PrincipalId []string
	// map[主账号id:子账号id]子账号名称
	PrincipalNames map[string]string
	// Read|ReadWrite|FullControl
	CannedAction string
	// 资源路径
	ResourcePath []string
	// 根据index 生成
	Id string
}

type SBucketPolicyStatementInput struct {
	// 主账号id:子账号id
	PrincipalId []string
	// Read|ReadWrite|FullControl
	CannedAction string
	// Allow|Deny
	Effect string
	// 被授权的资源地址,/*
	ResourcePath []string
	// ip 条件
	IpEquals    []string
	IpNotEquals []string
}

type SBucketMultipartUploads struct {
	// object name
	ObjectName string
	UploadID   string
	// 发起人
	Initiator string
	// 发起时间
	Initiated time.Time
}

type SBaseCloudObject struct {
	Key          string
	SizeBytes    int64
	StorageClass string
	ETag         string
	LastModified time.Time
	Meta         http.Header
}

type SListObjectResult struct {
	Objects        []ICloudObject
	NextMarker     string
	CommonPrefixes []ICloudObject
	IsTruncated    bool
}

type SGetObjectRange struct {
	Start int64
	End   int64
}

func (r SGetObjectRange) SizeBytes() int64 {
	return r.End - r.Start + 1
}

var (
	rangeExp = regexp.MustCompile(`(bytes=)?(\d*)-(\d*)`)
)

func ParseRange(rangeStr string) SGetObjectRange {
	objRange := SGetObjectRange{}
	if len(rangeStr) > 0 {
		find := rangeExp.FindAllStringSubmatch(rangeStr, -1)
		if len(find) > 0 && len(find[0]) > 3 {
			objRange.Start, _ = strconv.ParseInt(find[0][2], 10, 64)
			objRange.End, _ = strconv.ParseInt(find[0][3], 10, 64)
		}
	}
	return objRange
}

func (r SGetObjectRange) String() string {
	if r.Start > 0 && r.End > 0 {
		return fmt.Sprintf("bytes=%d-%d", r.Start, r.End)
	} else if r.Start > 0 && r.End <= 0 {
		return fmt.Sprintf("bytes=%d-", r.Start)
	} else if r.Start <= 0 && r.End > 0 {
		return fmt.Sprintf("bytes=0-%d", r.End)
	} else {
		return ""
	}
}

type ICloudBucket interface {
	IVirtualResource

	MaxPartCount() int
	MaxPartSizeBytes() int64

	//GetGlobalId() string
	//GetName() string
	GetAcl() TBucketACLType
	GetLocation() string
	GetIRegion() ICloudRegion
	GetStorageClass() string
	GetAccessUrls() []SBucketAccessUrl
	GetStats() SBucketStats
	GetLimit() SBucketStats
	SetLimit(limit SBucketStats) error
	LimitSupport() SBucketStats

	SetAcl(acl TBucketACLType) error

	ListObjects(prefix string, marker string, delimiter string, maxCount int) (SListObjectResult, error)

	CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl TBucketACLType, storageClassStr string, meta http.Header) error
	GetObject(ctx context.Context, key string, rangeOpt *SGetObjectRange) (io.ReadCloser, error)

	DeleteObject(ctx context.Context, keys string) error
	GetTempUrl(method string, key string, expire time.Duration) (string, error)

	PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl TBucketACLType, storageClassStr string, meta http.Header) error

	NewMultipartUpload(ctx context.Context, key string, cannedAcl TBucketACLType, storageClassStr string, meta http.Header) (string, error)
	UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error)
	CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucketName string, srcKey string, srcOffset int64, srcLength int64) (string, error)
	CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error
	AbortMultipartUpload(ctx context.Context, key string, uploadId string) error

	SetWebsite(conf SBucketWebsiteConf) error
	GetWebsiteConf() (SBucketWebsiteConf, error)
	DeleteWebSiteConf() error

	SetCORS(rules []SBucketCORSRule) error
	GetCORSRules() ([]SBucketCORSRule, error)
	DeleteCORS() error

	SetReferer(conf SBucketRefererConf) error
	GetReferer() (SBucketRefererConf, error)

	GetCdnDomains() ([]SCdnDomain, error)

	GetPolicy() ([]SBucketPolicyStatement, error)
	SetPolicy(policy SBucketPolicyStatementInput) error
	DeletePolicy(id []string) ([]SBucketPolicyStatement, error)

	ListMultipartUploads() ([]SBucketMultipartUploads, error)
}

type ICloudObject interface {
	GetIBucket() ICloudBucket

	GetKey() string
	GetSizeBytes() int64
	GetLastModified() time.Time
	GetStorageClass() string
	GetETag() string

	GetMeta() http.Header
	SetMeta(ctx context.Context, meta http.Header) error

	GetAcl() TBucketACLType
	SetAcl(acl TBucketACLType) error
}

type SCloudObject struct {
	Key          string
	SizeBytes    int64
	StorageClass string
	ETag         string
	LastModified time.Time
	Meta         http.Header
	Acl          string
}

func ICloudObject2Struct(obj ICloudObject) SCloudObject {
	return SCloudObject{
		Key:          obj.GetKey(),
		SizeBytes:    obj.GetSizeBytes(),
		StorageClass: obj.GetStorageClass(),
		ETag:         obj.GetETag(),
		LastModified: obj.GetLastModified(),
		Meta:         obj.GetMeta(),
		Acl:          string(obj.GetAcl()),
	}
}

func ICloudObject2JSONObject(obj ICloudObject) jsonutils.JSONObject {
	return jsonutils.Marshal(ICloudObject2Struct(obj))
}

func (o *SBaseCloudObject) GetKey() string {
	return o.Key
}

func (o *SBaseCloudObject) GetSizeBytes() int64 {
	return o.SizeBytes
}

func (o *SBaseCloudObject) GetLastModified() time.Time {
	return o.LastModified
}

func (o *SBaseCloudObject) GetStorageClass() string {
	return o.StorageClass
}

func (o *SBaseCloudObject) GetETag() string {
	return o.ETag
}

func (o *SBaseCloudObject) GetMeta() http.Header {
	return o.Meta
}

//func (o *SBaseCloudObject) SetMeta(meta http.Header) error {
//    return nil
//}

func GetIBucketById(region ICloudRegion, name string) (ICloudBucket, error) {
	buckets, err := region.GetIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIBuckets")
	}
	for i := range buckets {
		if buckets[i].GetGlobalId() == name {
			return buckets[i], nil
		}
	}
	return nil, ErrNotFound
}

func GetIBucketByName(region ICloudRegion, name string) (ICloudBucket, error) {
	buckets, err := region.GetIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIBuckets")
	}
	for i := range buckets {
		if buckets[i].GetName() == name {
			return buckets[i], nil
		}
	}
	return nil, ErrNotFound
}

func GetIBucketStats(bucket ICloudBucket) (SBucketStats, error) {
	stats := SBucketStats{
		ObjectCount: -1,
		SizeBytes:   -1,
	}
	objs, err := bucket.ListObjects("", "", "", 1000)
	if err != nil {
		return stats, errors.Wrap(err, "GetIObjects")
	}
	if objs.IsTruncated {
		return stats, errors.Wrap(ErrTooLarge, "too many objects")
	}
	stats.ObjectCount, stats.SizeBytes = 0, 0
	for _, obj := range objs.Objects {
		stats.SizeBytes += obj.GetSizeBytes()
		stats.ObjectCount += 1
	}
	return stats, nil
}

type cloudObjectList []ICloudObject

func (a cloudObjectList) Len() int           { return len(a) }
func (a cloudObjectList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a cloudObjectList) Less(i, j int) bool { return a[i].GetKey() < a[j].GetKey() }

func GetPagedObjects(bucket ICloudBucket, objectPrefix string, isRecursive bool, marker string, maxCount int) ([]ICloudObject, string, error) {
	delimiter := "/"
	if isRecursive {
		delimiter = ""
	}
	if maxCount > 1000 || maxCount <= 0 {
		maxCount = 1000
	}
	ret := make([]ICloudObject, 0)
	result, err := bucket.ListObjects(objectPrefix, marker, delimiter, maxCount)
	if err != nil {
		return nil, "", errors.Wrap(err, "bucket.ListObjects")
	}
	// Send all objects
	for i := range result.Objects {
		// if delimited, skip the first object ends with delimiter
		if !isRecursive && result.Objects[i].GetKey() == objectPrefix && strings.HasSuffix(objectPrefix, delimiter) {
			continue
		}
		ret = append(ret, result.Objects[i])
		marker = result.Objects[i].GetKey()
	}
	// Send all common prefixes if any.
	// NOTE: prefixes are only present if the request is delimited.
	if len(result.CommonPrefixes) > 0 {
		ret = append(ret, result.CommonPrefixes...)
	}
	// sort prefix by name in ascending order
	sort.Sort(cloudObjectList(ret))
	// If next marker present, save it for next request.
	if result.NextMarker != "" {
		marker = result.NextMarker
	}
	// If not truncated, no more objects
	if !result.IsTruncated {
		marker = ""
	}
	return ret, marker, nil
}

func GetAllObjects(bucket ICloudBucket, objectPrefix string, isRecursive bool) ([]ICloudObject, error) {
	ret := make([]ICloudObject, 0)
	// Save marker for next request.
	var marker string
	for {
		// Get list of objects a maximum of 1000 per request.
		result, marker, err := GetPagedObjects(bucket, objectPrefix, isRecursive, marker, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "bucket.ListObjects")
		}
		ret = append(ret, result...)
		if marker == "" {
			break
		}
	}
	return ret, nil
}

func GetIObject(bucket ICloudBucket, objectPrefix string) (ICloudObject, error) {
	tryPrefix := []string{objectPrefix}
	if strings.HasSuffix(objectPrefix, "/") {
		tryPrefix = append(tryPrefix, objectPrefix[:len(objectPrefix)-1])
	}
	for _, pref := range tryPrefix {
		result, err := bucket.ListObjects(pref, "", "", 1)
		if err != nil {
			return nil, errors.Wrap(err, "bucket.ListObjects")
		}
		objects := result.Objects
		if len(objects) > 0 && objects[0].GetKey() == objectPrefix {
			return objects[0], nil
		}
	}
	return nil, ErrNotFound
}

func Makedir(ctx context.Context, bucket ICloudBucket, key string) error {
	segs := make([]string, 0)
	for _, seg := range strings.Split(key, "/") {
		if len(seg) > 0 {
			segs = append(segs, seg)
		}
	}
	path := strings.Join(segs, "/") + "/"
	err := bucket.PutObject(ctx, path, strings.NewReader(""), 0, bucket.GetAcl(), "", nil)
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}

func UploadObject(ctx context.Context, bucket ICloudBucket, key string, blocksz int64, input io.Reader, sizeBytes int64, cannedAcl TBucketACLType, storageClass string, meta http.Header, debug bool) error {
	if blocksz <= 0 {
		blocksz = MAX_PUT_OBJECT_SIZEBYTES
	}
	if sizeBytes < blocksz {
		if debug {
			log.Debugf("too small, put object in one shot")
		}
		return bucket.PutObject(ctx, key, input, sizeBytes, cannedAcl, storageClass, meta)
	}
	partSize := blocksz
	partCount := sizeBytes / partSize
	if partCount*partSize < sizeBytes {
		partCount += 1
	}
	if partCount > int64(bucket.MaxPartCount()) {
		partCount = int64(bucket.MaxPartCount())
		partSize = sizeBytes / partCount
		if partSize*partCount < sizeBytes {
			partSize += 1
		}
		if partSize > bucket.MaxPartSizeBytes() {
			return errors.Error("too larget object")
		}
	}
	if debug {
		log.Debugf("multipart upload part count %d part size %d", partCount, partSize)
	}
	uploadId, err := bucket.NewMultipartUpload(ctx, key, cannedAcl, storageClass, meta)
	if err != nil {
		return errors.Wrap(err, "bucket.NewMultipartUpload")
	}
	etags := make([]string, partCount)
	offset := int64(0)
	for i := 0; i < int(partCount); i += 1 {
		if i == int(partCount)-1 {
			partSize = sizeBytes - partSize*(partCount-1)
		}
		if debug {
			log.Debugf("UploadPart %d %d", i+1, partSize)
		}
		etag, err := bucket.UploadPart(ctx, key, uploadId, i+1, io.LimitReader(input, partSize), partSize, offset, sizeBytes)
		if err != nil {
			err2 := bucket.AbortMultipartUpload(ctx, key, uploadId)
			if err2 != nil {
				log.Errorf("bucket.AbortMultipartUpload error %s", err2)
			}
			return errors.Wrap(err, "bucket.UploadPart")
		}
		offset += partSize
		etags[i] = etag
	}
	err = bucket.CompleteMultipartUpload(ctx, key, uploadId, etags)
	if err != nil {
		err2 := bucket.AbortMultipartUpload(ctx, key, uploadId)
		if err2 != nil {
			log.Errorf("bucket.AbortMultipartUpload error %s", err2)
		}
		return errors.Wrap(err, "CompleteMultipartUpload")
	}
	return nil
}

func DeletePrefix(ctx context.Context, bucket ICloudBucket, prefix string) error {
	objs, err := GetAllObjects(bucket, prefix, true)
	if err != nil {
		return errors.Wrap(err, "bucket.GetIObjects")
	}
	for i := range objs {
		err := bucket.DeleteObject(ctx, objs[i].GetKey())
		if err != nil {
			return errors.Wrap(err, "bucket.DeleteObject")
		}
	}
	return nil
}

func MergeMeta(src http.Header, dst http.Header) http.Header {
	if src != nil && dst != nil {
		ret := http.Header{}
		for k, vs := range src {
			for _, v := range vs {
				ret.Add(k, v)
			}
		}
		for k, vs := range dst {
			for _, v := range vs {
				ret.Add(k, v)
			}
		}
		return ret
	} else if src != nil && dst == nil {
		return src
	} else if src == nil && dst != nil {
		return dst
	} else {
		return nil
	}
}

func CopyObject(ctx context.Context, blocksz int64, dstBucket ICloudBucket, dstKey string, srcBucket ICloudBucket, srcKey string, dstMeta http.Header, debug bool) error {

	srcObj, err := GetIObject(srcBucket, srcKey)
	if err != nil {
		return errors.Wrap(err, "GetIObject")
	}
	if blocksz <= 0 {
		blocksz = MAX_PUT_OBJECT_SIZEBYTES
	}
	sizeBytes := srcObj.GetSizeBytes()
	if sizeBytes < blocksz {
		if debug {
			log.Debugf("too small, copy object in one shot")
		}
		srcStream, err := srcBucket.GetObject(ctx, srcKey, nil)
		if err != nil {
			return errors.Wrap(err, "srcBucket.GetObject")
		}
		defer srcStream.Close()
		err = dstBucket.PutObject(ctx, dstKey, srcStream, sizeBytes, srcObj.GetAcl(), srcObj.GetStorageClass(), MergeMeta(srcObj.GetMeta(), dstMeta))
		if err != nil {
			return errors.Wrap(err, "dstBucket.PutObject")
		}
		return nil
	}
	partSize := blocksz
	partCount := sizeBytes / partSize
	if partCount*partSize < sizeBytes {
		partCount += 1
	}
	if partCount > int64(dstBucket.MaxPartCount()) {
		partCount = int64(dstBucket.MaxPartCount())
		partSize = sizeBytes / partCount
		if partSize*partCount < sizeBytes {
			partSize += 1
		}
		if partSize > dstBucket.MaxPartSizeBytes() {
			return errors.Error("too larget object")
		}
	}
	if debug {
		log.Debugf("multipart upload part count %d part size %d", partCount, partSize)
	}
	uploadId, err := dstBucket.NewMultipartUpload(ctx, dstKey, srcObj.GetAcl(), srcObj.GetStorageClass(), MergeMeta(srcObj.GetMeta(), dstMeta))
	if err != nil {
		return errors.Wrap(err, "bucket.NewMultipartUpload")
	}
	etags := make([]string, partCount)
	offset := int64(0)
	for i := 0; i < int(partCount); i += 1 {
		start := int64(i) * partSize
		if i == int(partCount)-1 {
			partSize = sizeBytes - partSize*(partCount-1)
		}
		end := start + partSize - 1
		rangeOpt := SGetObjectRange{
			Start: start,
			End:   end,
		}
		if debug {
			log.Debugf("UploadPart %d %d range: %s (%d)", i+1, partSize, rangeOpt.String(), rangeOpt.SizeBytes())
		}
		srcStream, err := srcBucket.GetObject(ctx, srcKey, &rangeOpt)
		if err == nil {
			defer srcStream.Close()
			var etag string
			etag, err = dstBucket.UploadPart(ctx, dstKey, uploadId, i+1, io.LimitReader(srcStream, partSize), partSize, offset, sizeBytes)
			if err == nil {
				etags[i] = etag
				continue
			}
		}
		offset += partSize
		if err != nil {
			err2 := dstBucket.AbortMultipartUpload(ctx, dstKey, uploadId)
			if err2 != nil {
				log.Errorf("bucket.AbortMultipartUpload error %s", err2)
			}
			return errors.Wrap(err, "bucket.UploadPart")
		}
	}
	err = dstBucket.CompleteMultipartUpload(ctx, dstKey, uploadId, etags)
	if err != nil {
		err2 := dstBucket.AbortMultipartUpload(ctx, dstKey, uploadId)
		if err2 != nil {
			log.Errorf("bucket.AbortMultipartUpload error %s", err2)
		}
		return errors.Wrap(err, "CompleteMultipartUpload")
	}
	return nil
}

func CopyPart(ctx context.Context,
	iDstBucket ICloudBucket, dstKey string, uploadId string, partNumber int,
	iSrcBucket ICloudBucket, srcKey string, rangeOpt *SGetObjectRange,
) (string, error) {
	srcReader, err := iSrcBucket.GetObject(ctx, srcKey, rangeOpt)
	if err != nil {
		return "", errors.Wrap(err, "iSrcBucket.GetObject")
	}
	defer srcReader.Close()

	etag, err := iDstBucket.UploadPart(ctx, dstKey, uploadId, partNumber, io.LimitReader(srcReader, rangeOpt.SizeBytes()), rangeOpt.SizeBytes(), 0, 0)
	if err != nil {
		return "", errors.Wrap(err, "iDstBucket.UploadPart")
	}
	return etag, nil
}

func ObjectSetMeta(ctx context.Context,
	bucket ICloudBucket, obj ICloudObject,
	meta http.Header,
) error {
	return bucket.CopyObject(ctx, obj.GetKey(), bucket.GetName(), obj.GetKey(), obj.GetAcl(), obj.GetStorageClass(), meta)
}

func MetaToHttpHeader(metaPrefix string, meta http.Header) http.Header {
	hdr := http.Header{}
	for k, v := range meta {
		if len(v) == 0 || len(v[0]) == 0 {
			continue
		}
		k = http.CanonicalHeaderKey(k)
		switch k {
		case META_HEADER_CACHE_CONTROL,
			META_HEADER_CONTENT_TYPE,
			META_HEADER_CONTENT_DISPOSITION,
			META_HEADER_CONTENT_ENCODING,
			META_HEADER_CONTENT_LANGUAGE,
			META_HEADER_CONTENT_MD5:
			hdr.Set(k, v[0])
		default:
			hdr.Set(fmt.Sprintf("%s%s", metaPrefix, k), v[0])
		}
	}
	return hdr
}

func FetchMetaFromHttpHeader(metaPrefix string, headers http.Header) http.Header {
	metaPrefix = http.CanonicalHeaderKey(metaPrefix)
	meta := http.Header{}
	for hdr, vals := range headers {
		hdr = http.CanonicalHeaderKey(hdr)
		if strings.HasPrefix(hdr, metaPrefix) {
			for _, val := range vals {
				meta.Add(hdr[len(metaPrefix):], val)
			}
		}
	}
	for _, hdr := range []string{
		META_HEADER_CONTENT_TYPE,
		META_HEADER_CONTENT_ENCODING,
		META_HEADER_CONTENT_DISPOSITION,
		META_HEADER_CONTENT_LANGUAGE,
		META_HEADER_CACHE_CONTROL,
	} {
		val := headers.Get(hdr)
		if len(val) > 0 {
			meta.Set(hdr, val)
		}
	}
	return meta
}

func SetBucketCORS(ibucket ICloudBucket, rules []SBucketCORSRule) error {
	if len(rules) == 0 {
		return nil
	}

	oldRules, err := ibucket.GetCORSRules()
	if err != nil {
		return errors.Wrap(err, "ibucket.GetCORSRules()")
	}

	newSet := []SBucketCORSRule{}
	updateSet := map[int]SBucketCORSRule{}
	for i := range rules {
		index, err := strconv.Atoi(rules[i].Id)
		if err == nil && index < len(oldRules) {
			updateSet[index] = rules[i]
		} else {
			newSet = append(newSet, rules[i])
		}
	}

	updatedRules := []SBucketCORSRule{}
	for i := range oldRules {
		if _, ok := updateSet[i]; !ok {
			updatedRules = append(updatedRules, oldRules[i])
		} else {
			updatedRules = append(updatedRules, updateSet[i])
		}
	}
	updatedRules = append(updatedRules, newSet...)

	err = ibucket.SetCORS(updatedRules)
	if err != nil {
		return errors.Wrap(err, "ibucket.SetCORS(updatedRules)")
	}
	return nil
}

func DeleteBucketCORS(ibucket ICloudBucket, id []string) ([]SBucketCORSRule, error) {
	if len(id) == 0 {
		return nil, nil
	}
	deletedRules := []SBucketCORSRule{}

	oldRules, err := ibucket.GetCORSRules()
	if err != nil {
		return nil, errors.Wrap(err, "ibucket.GetCORSRules()")
	}

	excludeMap := map[int]bool{}
	for i := range id {
		index, err := strconv.Atoi(id[i])
		if err == nil && index < len(oldRules) {
			excludeMap[index] = true
		}
	}
	if len(excludeMap) == 0 {
		return nil, nil
	}

	newRules := []SBucketCORSRule{}
	for i := range oldRules {
		if _, ok := excludeMap[i]; !ok {
			newRules = append(newRules, oldRules[i])
		} else {
			deletedRules = append(deletedRules, oldRules[i])
		}
	}

	if len(newRules) == 0 {
		err = ibucket.DeleteCORS()
		if err != nil {
			return nil, errors.Wrapf(err, "ibucket.DeleteCORS()")
		}
	} else {
		err = ibucket.SetCORS(newRules)
		if err != nil {
			return nil, errors.Wrapf(err, "ibucket.SetBucketCORS(newRules)")
		}
	}

	return deletedRules, nil
}

func SetBucketTags(ctx context.Context, iBucket ICloudBucket, mangerId string, tags map[string]string) (TagsUpdateInfo, error) {
	ret := TagsUpdateInfo{}
	old, err := iBucket.GetTags()
	if err != nil {
		if errors.Cause(err) == ErrNotImplemented || errors.Cause(err) == ErrNotSupported {
			return ret, nil
		}
		return ret, errors.Wrapf(err, "iBucket.GetTags")
	}
	ret.OldTags, ret.NewTags = old, tags
	if !ret.IsChanged() {
		return ret, nil
	}
	return ret, SetTags(ctx, iBucket, mangerId, tags, true)
}
