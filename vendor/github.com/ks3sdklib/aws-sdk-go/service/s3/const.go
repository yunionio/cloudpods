package s3

// HTTP headers
const (
	HTTPHeaderAcceptEncoding            string = "Accept-Encoding"
	HTTPHeaderAuthorization                    = "Authorization"
	HTTPHeaderCacheControl                     = "Cache-Control"
	HTTPHeaderContentDisposition               = "Content-Disposition"
	HTTPHeaderContentEncoding                  = "Content-Encoding"
	HTTPHeaderContentLength                    = "Content-Length"
	HTTPHeaderContentMD5                       = "Content-MD5"
	HTTPHeaderContentType                      = "Content-Type"
	HTTPHeaderContentLanguage                  = "Content-Language"
	HTTPHeaderLastModified                     = "Last-Modified"
	HTTPHeaderDate                             = "Date"
	HTTPHeaderEtag                             = "Etag"
	HTTPHeaderExpires                          = "Expires"
	HTTPHeaderHost                             = "Host"
	HTTPHeaderAmzACL                           = "X-Amz-Acl"
	HTTPHeaderAmzChecksumCrc64ecma             = "X-Amz-Checksum-Crc64ecma"
	HTTPHeaderAmzStorageClass                  = "X-Amz-Storage-Class"
	HTTPHeaderAmzDataRedundancyType            = "X-Amz-Data-Redundancy-Type"
	HTTPHeaderAmzZRSSwitchEnable               = "X-Amz-Zrs-Switch-Enable"
	HTTPHeaderAmzAllowSameActionOverlap        = "X-Amz-Allow-Same-Action-Overlap"
	HTTPHeaderAmzBucketType                    = "X-Amz-Bucket-Type"
	HTTPHeaderAmzBucketVisitType               = "X-Amz-Bucket-Visit-Type"
)

// ACL
const (
	ACLPrivate         string = "private"
	ACLPublicRead      string = "public-read"
	ACLPublicReadWrite string = "public-read-write"
)

// StorageClass
const (
	StorageClassExtremePL3      string = "EXTREME_PL3"
	StorageClassExtremePL2      string = "EXTREME_PL2"
	StorageClassExtremePL1      string = "EXTREME_PL1"
	StorageClassStandard        string = "STANDARD"
	StorageClassIA              string = "STANDARD_IA"
	StorageClassDeepIA          string = "DEEP_IA"
	StorageClassArchive         string = "ARCHIVE"
	StorageClassDeepColdArchive string = "DEEP_COLD_ARCHIVE"
)

// BucketType
const (
	BucketTypeExtremePL3 string = "EXTREME_PL3"
	BucketTypeExtremePL2 string = "EXTREME_PL2"
	BucketTypeExtremePL1 string = "EXTREME_PL1"
	BucketTypeNormal     string = "NORMAL"
	BucketTypeIA         string = "IA"
	BucketTypeDeepIA     string = "DEEP_IA"
	BucketTypeArchive    string = "ARCHIVE"
)

const (
	BucketVisitTypeNormal       string = "NORMAL"
	BucketVisitTypeFrequentList string = "FREQUENTLIST"
)

type HTTPMethod string

const (
	PUT    HTTPMethod = "PUT"
	GET    HTTPMethod = "GET"
	DELETE HTTPMethod = "DELETE"
	HEAD   HTTPMethod = "HEAD"
	POST   HTTPMethod = "POST"
)

const (
	AllUsersUri = "http://acs.amazonaws.com/groups/global/AllUsers"
	MetaPrefix  = "x-amz-meta-"
)

const (
	DataRedundancyTypeLRS string = "LRS"
	DataRedundancyTypeZRS string = "ZRS"
)

const (
	StorageMediumNormal  string = "Normal"
	StorageMediumExtreme string = "Extreme"
)

const (
	AlgorithmAES256 string = "AES256"
	AlgorithmSM4    string = "SM4"
)

const (
	StatusEnabled  string = "Enabled"
	StatusDisabled string = "Disabled"
)
