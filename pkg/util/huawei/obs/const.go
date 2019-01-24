package obs

const (
	obs_sdk_version        = "3.0.0"
	USER_AGENT             = "obs-sdk-go/" + obs_sdk_version
	HEADER_PREFIX          = "x-amz-"
	HEADER_PREFIX_META     = "x-amz-meta-"
	HEADER_PREFIX_OBS      = "x-obs-"
	HEADER_PREFIX_META_OBS = "x-obs-meta-"
	HEADER_DATE_AMZ        = "x-amz-date"
	HEADER_DATE_OBS        = "x-obs-date"
	HEADER_STS_TOKEN_AMZ   = "x-amz-security-token"
	HEADER_ACCESSS_KEY_AMZ = "AWSAccessKeyId"
	PREFIX_META            = "meta-"

	HEADER_CONTENT_SHA256_AMZ               = "x-amz-content-sha256"
	HEADER_ACL_AMZ                          = "x-amz-acl"
	HEADER_ACL_OBS                          = "x-obs-acl"
	HEADER_ACL                              = "acl"
	HEADER_LOCATION_AMZ                     = "location"
	HEADER_BUCKET_LOCATION_OBS              = "bucket-location"
	HEADER_COPY_SOURCE                      = "copy-source"
	HEADER_COPY_SOURCE_RANGE                = "copy-source-range"
	HEADER_RANGE                            = "Range"
	HEADER_STORAGE_CLASS                    = "x-default-storage-class"
	HEADER_STORAGE_CLASS_OBS                = "x-obs-storage-class"
	HEADER_VERSION_OBS                      = "version"
	HEADER_GRANT_READ_OBS                   = "grant-read"
	HEADER_GRANT_WRITE_OBS                  = "grant-write"
	HEADER_GRANT_READ_ACP_OBS               = "grant-read-acp"
	HEADER_GRANT_WRITE_ACP_OBS              = "grant-write-acp"
	HEADER_GRANT_FULL_CONTROL_OBS           = "grant-full-control"
	HEADER_GRANT_READ_DELIVERED_OBS         = "grant-read-delivered"
	HEADER_GRANT_FULL_CONTROL_DELIVERED_OBS = "grant-full-control-delivered"
	HEADER_REQUEST_ID                       = "request-id"
	HEADER_BUCKET_REGION                    = "bucket-region"
	HEADER_ACCESS_CONRTOL_ALLOW_ORIGIN      = "access-control-allow-origin"
	HEADER_ACCESS_CONRTOL_ALLOW_HEADERS     = "access-control-allow-headers"
	HEADER_ACCESS_CONRTOL_MAX_AGE           = "access-control-max-age"
	HEADER_ACCESS_CONRTOL_ALLOW_METHODS     = "access-control-allow-methods"
	HEADER_ACCESS_CONRTOL_EXPOSE_HEADERS    = "access-control-expose-headers"
	HEADER_EPID_HEADERS                     = "epid"
	HEADER_VERSION_ID                       = "version-id"
	HEADER_COPY_SOURCE_VERSION_ID           = "copy-source-version-id"
	HEADER_DELETE_MARKER                    = "delete-marker"
	HEADER_WEBSITE_REDIRECT_LOCATION        = "website-redirect-location"
	HEADER_METADATA_DIRECTIVE               = "metadata-directive"
	HEADER_EXPIRATION                       = "expiration"
	HEADER_EXPIRES_OBS                      = "x-obs-expires"
	HEADER_RESTORE                          = "restore"
	HEADER_OBJECT_TYPE                      = "object-type"
	HEADER_NEXT_APPEND_POSITION             = "next-append-position"
	HEADER_STORAGE_CLASS2                   = "storage-class"
	HEADER_CONTENT_LENGTH                   = "content-length"
	HEADER_CONTENT_TYPE                     = "content-type"
	HEADER_CONTENT_LANGUAGE                 = "content-language"
	HEADER_EXPIRES                          = "expires"
	HEADER_CACHE_CONTROL                    = "cache-control"
	HEADER_CONTENT_DISPOSITION              = "content-disposition"
	HEADER_CONTENT_ENCODING                 = "content-encoding"

	HEADER_ETAG         = "etag"
	HEADER_LASTMODIFIED = "last-modified"

	HEADER_COPY_SOURCE_IF_MATCH            = "copy-source-if-match"
	HEADER_COPY_SOURCE_IF_NONE_MATCH       = "copy-source-if-none-match"
	HEADER_COPY_SOURCE_IF_MODIFIED_SINCE   = "copy-source-if-modified-since"
	HEADER_COPY_SOURCE_IF_UNMODIFIED_SINCE = "copy-source-if-unmodified-since"

	HEADER_IF_MATCH            = "If-Match"
	HEADER_IF_NONE_MATCH       = "If-None-Match"
	HEADER_IF_MODIFIED_SINCE   = "If-Modified-Since"
	HEADER_IF_UNMODIFIED_SINCE = "If-Unmodified-Since"

	HEADER_SSEC_ENCRYPTION = "server-side-encryption-customer-algorithm"
	HEADER_SSEC_KEY        = "server-side-encryption-customer-key"
	HEADER_SSEC_KEY_MD5    = "server-side-encryption-customer-key-MD5"

	HEADER_SSEKMS_ENCRYPTION      = "server-side-encryption"
	HEADER_SSEKMS_KEY             = "server-side-encryption-aws-kms-key-id"
	HEADER_SSEKMS_ENCRYPT_KEY_OBS = "server-side-encryption-kms-key-id"

	HEADER_SSEC_COPY_SOURCE_ENCRYPTION = "copy-source-server-side-encryption-customer-algorithm"
	HEADER_SSEC_COPY_SOURCE_KEY        = "copy-source-server-side-encryption-customer-key"
	HEADER_SSEC_COPY_SOURCE_KEY_MD5    = "copy-source-server-side-encryption-customer-key-MD5"

	HEADER_SSEKMS_KEY_AMZ = "x-amz-server-side-encryption-aws-kms-key-id"

	HEADER_SSEKMS_KEY_OBS = "x-obs-server-side-encryption-kms-key-id"

	HEADER_SUCCESS_ACTION_REDIRECT = "success_action_redirect"

	HEADER_DATE_CAMEL                          = "Date"
	HEADER_HOST_CAMEL                          = "Host"
	HEADER_HOST                                = "host"
	HEADER_AUTH_CAMEL                          = "Authorization"
	HEADER_MD5_CAMEL                           = "Content-MD5"
	HEADER_LOCATION_CAMEL                      = "Location"
	HEADER_CONTENT_LENGTH_CAMEL                = "Content-Length"
	HEADER_CONTENT_TYPE_CAML                   = "Content-Type"
	HEADER_USER_AGENT_CAMEL                    = "User-Agent"
	HEADER_ORIGIN_CAMEL                        = "Origin"
	HEADER_ACCESS_CONTROL_REQUEST_HEADER_CAMEL = "Access-Control-Request-Headers"

	PARAM_VERSION_ID                   = "versionId"
	PARAM_RESPONSE_CONTENT_TYPE        = "response-content-type"
	PARAM_RESPONSE_CONTENT_LANGUAGE    = "response-content-language"
	PARAM_RESPONSE_EXPIRES             = "response-expires"
	PARAM_RESPONSE_CACHE_CONTROL       = "response-cache-control"
	PARAM_RESPONSE_CONTENT_DISPOSITION = "response-content-disposition"
	PARAM_RESPONSE_CONTENT_ENCODING    = "response-content-encoding"
	PARAM_IMAGE_PROCESS                = "x-image-process"

	PARAM_ALGORITHM_AMZ_CAMEL     = "X-Amz-Algorithm"
	PARAM_CREDENTIAL_AMZ_CAMEL    = "X-Amz-Credential"
	PARAM_DATE_AMZ_CAMEL          = "X-Amz-Date"
	PARAM_DATE_OBS_CAMEL          = "X-Obs-Date"
	PARAM_EXPIRES_AMZ_CAMEL       = "X-Amz-Expires"
	PARAM_SIGNEDHEADERS_AMZ_CAMEL = "X-Amz-SignedHeaders"
	PARAM_SIGNATURE_AMZ_CAMEL     = "X-Amz-Signature"

	DEFAULT_SIGNATURE            = SignatureV2
	DEFAULT_REGION               = "region"
	DEFAULT_CONNECT_TIMEOUT      = 60
	DEFAULT_SOCKET_TIMEOUT       = 60
	DEFAULT_HEADER_TIMEOUT       = 60
	DEFAULT_IDLE_CONN_TIMEOUT    = 30
	DEFAULT_MAX_RETRY_COUNT      = 3
	DEFAULT_MAX_CONN_PER_HOST    = 1000
	EMPTY_CONTENT_SHA256         = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	UNSIGNED_PAYLOAD             = "UNSIGNED-PAYLOAD"
	LONG_DATE_FORMAT             = "20060102T150405Z"
	SHORT_DATE_FORMAT            = "20060102"
	ISO8601_DATE_FORMAT          = "2006-01-02T15:04:05Z"
	ISO8601_MIDNIGHT_DATE_FORMAT = "2006-01-02T00:00:00Z"
	RFC1123_FORMAT               = "Mon, 02 Jan 2006 15:04:05 GMT"

	V4_SERVICE_NAME   = "s3"
	V4_SERVICE_SUFFIX = "aws4_request"

	V2_HASH_PREFIX  = "AWS"
	OBS_HASH_PREFIX = "OBS"

	V4_HASH_PREFIX = "AWS4-HMAC-SHA256"
	V4_HASH_PRE    = "AWS4"

	DEFAULT_SSE_KMS_ENCRYPTION     = "aws:kms"
	DEFAULT_SSE_KMS_ENCRYPTION_OBS = "kms"

	DEFAULT_SSE_C_ENCRYPTION = "AES256"

	HTTP_GET     = "GET"
	HTTP_POST    = "POST"
	HTTP_PUT     = "PUT"
	HTTP_DELETE  = "DELETE"
	HTTP_HEAD    = "HEAD"
	HTTP_OPTIONS = "OPTIONS"
)

type SignatureType string

const (
	SignatureV2  SignatureType = "v2"
	SignatureV4  SignatureType = "v4"
	SignatureObs SignatureType = "OBS"
)

var (
	interested_headers = []string{"content-md5", "content-type", "date"}

	allowed_response_http_header_metadata_names = map[string]bool{
		"content-type":                  true,
		"content-md5":                   true,
		"content-length":                true,
		"content-language":              true,
		"expires":                       true,
		"origin":                        true,
		"cache-control":                 true,
		"content-disposition":           true,
		"content-encoding":              true,
		"x-default-storage-class":       true,
		"location":                      true,
		"date":                          true,
		"etag":                          true,
		"host":                          true,
		"last-modified":                 true,
		"content-range":                 true,
		"x-reserved":                    true,
		"access-control-allow-origin":   true,
		"access-control-allow-headers":  true,
		"access-control-max-age":        true,
		"access-control-allow-methods":  true,
		"access-control-expose-headers": true,
		"connection":                    true,
	}

	allowed_request_http_header_metadata_names = map[string]bool{
		"content-type":                   true,
		"content-md5":                    true,
		"content-length":                 true,
		"content-language":               true,
		"expires":                        true,
		"origin":                         true,
		"cache-control":                  true,
		"content-disposition":            true,
		"content-encoding":               true,
		"access-control-request-method":  true,
		"access-control-request-headers": true,
		"x-default-storage-class":        true,
		"location":                       true,
		"date":                           true,
		"etag":                           true,
		"range":                          true,
		"host":                           true,
		"if-modified-since":              true,
		"if-unmodified-since":            true,
		"if-match":                       true,
		"if-none-match":                  true,
		"last-modified":                  true,
		"content-range":                  true,
	}

	allowed_resource_parameter_names = map[string]bool{
		"acl":                          true,
		"backtosource":                 true,
		"policy":                       true,
		"torrent":                      true,
		"logging":                      true,
		"location":                     true,
		"storageinfo":                  true,
		"quota":                        true,
		"storageclass":                 true,
		"storagepolicy":                true,
		"requestpayment":               true,
		"versions":                     true,
		"versioning":                   true,
		"versionid":                    true,
		"uploads":                      true,
		"uploadid":                     true,
		"partnumber":                   true,
		"website":                      true,
		"notification":                 true,
		"lifecycle":                    true,
		"deletebucket":                 true,
		"delete":                       true,
		"cors":                         true,
		"restore":                      true,
		"tagging":                      true,
		"append":                       true,
		"position":                     true,
		"replication":                  true,
		"response-content-type":        true,
		"response-content-language":    true,
		"response-expires":             true,
		"response-cache-control":       true,
		"response-content-disposition": true,
		"response-content-encoding":    true,
		"x-image-process":              true,
		"x-oss-process":                true,
	}

	mime_types = map[string]string{
		"7z":      "application/x-7z-compressed",
		"aac":     "audio/x-aac",
		"ai":      "application/postscript",
		"aif":     "audio/x-aiff",
		"asc":     "text/plain",
		"asf":     "video/x-ms-asf",
		"atom":    "application/atom+xml",
		"avi":     "video/x-msvideo",
		"bmp":     "image/bmp",
		"bz2":     "application/x-bzip2",
		"cer":     "application/pkix-cert",
		"crl":     "application/pkix-crl",
		"crt":     "application/x-x509-ca-cert",
		"css":     "text/css",
		"csv":     "text/csv",
		"cu":      "application/cu-seeme",
		"deb":     "application/x-debian-package",
		"doc":     "application/msword",
		"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"dvi":     "application/x-dvi",
		"eot":     "application/vnd.ms-fontobject",
		"eps":     "application/postscript",
		"epub":    "application/epub+zip",
		"etx":     "text/x-setext",
		"flac":    "audio/flac",
		"flv":     "video/x-flv",
		"gif":     "image/gif",
		"gz":      "application/gzip",
		"htm":     "text/html",
		"html":    "text/html",
		"ico":     "image/x-icon",
		"ics":     "text/calendar",
		"ini":     "text/plain",
		"iso":     "application/x-iso9660-image",
		"jar":     "application/java-archive",
		"jpe":     "image/jpeg",
		"jpeg":    "image/jpeg",
		"jpg":     "image/jpeg",
		"js":      "text/javascript",
		"json":    "application/json",
		"latex":   "application/x-latex",
		"log":     "text/plain",
		"m4a":     "audio/mp4",
		"m4v":     "video/mp4",
		"mid":     "audio/midi",
		"midi":    "audio/midi",
		"mov":     "video/quicktime",
		"mp3":     "audio/mpeg",
		"mp4":     "video/mp4",
		"mp4a":    "audio/mp4",
		"mp4v":    "video/mp4",
		"mpe":     "video/mpeg",
		"mpeg":    "video/mpeg",
		"mpg":     "video/mpeg",
		"mpg4":    "video/mp4",
		"oga":     "audio/ogg",
		"ogg":     "audio/ogg",
		"ogv":     "video/ogg",
		"ogx":     "application/ogg",
		"pbm":     "image/x-portable-bitmap",
		"pdf":     "application/pdf",
		"pgm":     "image/x-portable-graymap",
		"png":     "image/png",
		"pnm":     "image/x-portable-anymap",
		"ppm":     "image/x-portable-pixmap",
		"ppt":     "application/vnd.ms-powerpoint",
		"pptx":    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"ps":      "application/postscript",
		"qt":      "video/quicktime",
		"rar":     "application/x-rar-compressed",
		"ras":     "image/x-cmu-raster",
		"rss":     "application/rss+xml",
		"rtf":     "application/rtf",
		"sgm":     "text/sgml",
		"sgml":    "text/sgml",
		"svg":     "image/svg+xml",
		"swf":     "application/x-shockwave-flash",
		"tar":     "application/x-tar",
		"tif":     "image/tiff",
		"tiff":    "image/tiff",
		"torrent": "application/x-bittorrent",
		"ttf":     "application/x-font-ttf",
		"txt":     "text/plain",
		"wav":     "audio/x-wav",
		"webm":    "video/webm",
		"wma":     "audio/x-ms-wma",
		"wmv":     "video/x-ms-wmv",
		"woff":    "application/x-font-woff",
		"wsdl":    "application/wsdl+xml",
		"xbm":     "image/x-xbitmap",
		"xls":     "application/vnd.ms-excel",
		"xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"xml":     "application/xml",
		"xpm":     "image/x-xpixmap",
		"xwd":     "image/x-xwindowdump",
		"yaml":    "text/yaml",
		"yml":     "text/yaml",
		"zip":     "application/zip",
	}
)

type HttpMethodType string

const (
	HttpMethodGet     HttpMethodType = HTTP_GET
	HttpMethodPut     HttpMethodType = HTTP_PUT
	HttpMethodPost    HttpMethodType = HTTP_POST
	HttpMethodDelete  HttpMethodType = HTTP_DELETE
	HttpMethodHead    HttpMethodType = HTTP_HEAD
	HttpMethodOptions HttpMethodType = HTTP_OPTIONS
)

type SubResourceType string

const (
	SubResourceStoragePolicy SubResourceType = "storagePolicy"
	SubResourceStorageClass  SubResourceType = "storageClass"
	SubResourceQuota         SubResourceType = "quota"
	SubResourceStorageInfo   SubResourceType = "storageinfo"
	SubResourceLocation      SubResourceType = "location"
	SubResourceAcl           SubResourceType = "acl"
	SubResourcePolicy        SubResourceType = "policy"
	SubResourceCors          SubResourceType = "cors"
	SubResourceVersioning    SubResourceType = "versioning"
	SubResourceWebsite       SubResourceType = "website"
	SubResourceLogging       SubResourceType = "logging"
	SubResourceLifecycle     SubResourceType = "lifecycle"
	SubResourceNotification  SubResourceType = "notification"
	SubResourceTagging       SubResourceType = "tagging"
	SubResourceDelete        SubResourceType = "delete"
	SubResourceVersions      SubResourceType = "versions"
	SubResourceUploads       SubResourceType = "uploads"
	SubResourceRestore       SubResourceType = "restore"
)

type AclType string

const (
	AclPrivate                 AclType = "private"
	AclPublicRead              AclType = "public-read"
	AclPublicReadWrite         AclType = "public-read-write"
	AclAuthenticatedRead       AclType = "authenticated-read"
	AclBucketOwnerRead         AclType = "bucket-owner-read"
	AclBucketOwnerFullControl  AclType = "bucket-owner-full-control"
	AclLogDeliveryWrite        AclType = "log-delivery-write"
	AclPublicReadDelivery      AclType = "public-read-delivered"
	AclPublicReadWriteDelivery AclType = "public-read-write-delivered"
)

type StorageClassType string

const (
	StorageClassStandard StorageClassType = "STANDARD"
	StorageClassWarm     StorageClassType = "STANDARD_IA"
	StorageClassCold     StorageClassType = "GLACIER"
)

type PermissionType string

const (
	PermissionRead        PermissionType = "READ"
	PermissionWrite       PermissionType = "WRITE"
	PermissionReadAcp     PermissionType = "READ_ACP"
	PermissionWriteAcp    PermissionType = "WRITE_ACP"
	PermissionFullControl PermissionType = "FULL_CONTROL"
)

type GranteeType string

const (
	GranteeGroup GranteeType = "Group"
	GranteeUser  GranteeType = "CanonicalUser"
)

type GroupUriType string

const (
	GroupAllUsers           GroupUriType = "AllUsers"
	GroupAuthenticatedUsers GroupUriType = "AuthenticatedUsers"
	GroupLogDelivery        GroupUriType = "LogDelivery"
)

type VersioningStatusType string

const (
	VersioningStatusEnabled   VersioningStatusType = "Enabled"
	VersioningStatusSuspended VersioningStatusType = "Suspended"
)

type ProtocolType string

const (
	ProtocolHttp  ProtocolType = "http"
	ProtocolHttps ProtocolType = "https"
)

type RuleStatusType string

const (
	RuleStatusEnabled  RuleStatusType = "Enabled"
	RuleStatusDisabled RuleStatusType = "Disabled"
)

type RestoreTierType string

const (
	RestoreTierExpedited RestoreTierType = "Expedited"
	RestoreTierStandard  RestoreTierType = "Standard"
	RestoreTierBulk      RestoreTierType = "Bulk"
)

type MetadataDirectiveType string

const (
	CopyMetadata    MetadataDirectiveType = "COPY"
	ReplaceMetadata MetadataDirectiveType = "REPLACE"
)

type EventType string

const (
	ObjectCreatedAll  EventType = "ObjectCreated:*"
	ObjectCreatedPut  EventType = "ObjectCreated:Put"
	ObjectCreatedPost EventType = "ObjectCreated:Post"

	ObjectCreatedCopy                    EventType = "ObjectCreated:Copy"
	ObjectCreatedCompleteMultipartUpload EventType = "ObjectCreated:CompleteMultipartUpload"
	ObjectRemovedAll                     EventType = "ObjectRemoved:*"
	ObjectRemovedDelete                  EventType = "ObjectRemoved:Delete"
	ObjectRemovedDeleteMarkerCreated     EventType = "ObjectRemoved:DeleteMarkerCreated"
)
