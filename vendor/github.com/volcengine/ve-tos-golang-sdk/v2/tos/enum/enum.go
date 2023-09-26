package enum

type ACLType string

const (
	ACLPrivate                ACLType = "private"
	ACLPublicRead             ACLType = "public-read"
	ACLPublicReadWrite        ACLType = "public-read-write"
	ACLAuthRead               ACLType = "authenticated-read"
	ACLBucketOwnerRead        ACLType = "bucket-owner-read"
	ACLBucketOwnerFullControl ACLType = "bucket-owner-full-control"
	ACLLogDeliveryWrite       ACLType = "log-delivery-write"
	ACLBucketOwnerEntrusted   ACLType = "bucket-owner-entrusted"
)

type StorageClassType string

const (
	StorageClassStandard           StorageClassType = "STANDARD"
	StorageClassIa                 StorageClassType = "IA"
	StorageClassArchiveFr          StorageClassType = "ARCHIVE_FR"
	StorageClassIntelligentTiering StorageClassType = "INTELLIGENT_TIERING"
	StorageClassColdArchive        StorageClassType = "COLD_ARCHIVE"
)

type MetadataDirectiveType string

const (
	// MetadataDirectiveReplace replace source object metadata when calling CopyObject
	MetadataDirectiveReplace MetadataDirectiveType = "REPLACE"

	// MetadataDirectiveCopy copy source object metadata when calling CopyObject
	MetadataDirectiveCopy MetadataDirectiveType = "COPY"
)

type AzRedundancyType string

const (
	AzRedundancySingleAz AzRedundancyType = "single-az"
	AzRedundancyMultiAz  AzRedundancyType = "multi-az"
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

type CannedType string

const (
	CannedAllUsers           CannedType = "AllUsers"
	CannedAuthenticatedUsers CannedType = "AuthenticatedUsers"
)

type DataTransferType int

const (
	DataTransferStarted DataTransferType = 1
	DataTransferRW      DataTransferType = 2
	DataTransferSucceed DataTransferType = 3
	DataTransferFailed  DataTransferType = 4
)

type HttpMethodType string

const (
	HttpMethodGet    HttpMethodType = "GET"
	HttpMethodPut    HttpMethodType = "PUT"
	HttpMethodPost   HttpMethodType = "POST"
	HttpMethodDelete HttpMethodType = "DELETE"
	HttpMethodHead   HttpMethodType = "HEAD"
)

type UploadEventType int

const (
	UploadEventCreateMultipartUploadSucceed   UploadEventType = 1
	UploadEventCreateMultipartUploadFailed    UploadEventType = 2
	UploadEventUploadPartSucceed              UploadEventType = 3
	UploadEventUploadPartFailed               UploadEventType = 4
	UploadEventUploadPartAborted              UploadEventType = 5 // The task needs to be interrupted in case of 403, 404, 405 errors
	UploadEventCompleteMultipartUploadSucceed UploadEventType = 6
	UploadEventCompleteMultipartUploadFailed  UploadEventType = 7
)

type DownloadEventType int

const (
	DownloadEventCreateTempFileSucceed DownloadEventType = 1
	DownloadEventCreateTempFileFailed  DownloadEventType = 2
	DownloadEventDownloadPartSucceed   DownloadEventType = 3
	DownloadEventDownloadPartFailed    DownloadEventType = 4
	DownloadEventDownloadPartAborted   DownloadEventType = 5 // The task needs to be interrupted in case of 403, 404, 405 errors
	DownloadEventRenameTempFileSucceed DownloadEventType = 6
	DownloadEventRenameTempFileFailed  DownloadEventType = 7
)

type CertStatusType string

const (
	CertStatusBound   CertStatusType = "CertBound"
	CertStatusUnbound CertStatusType = "CertUnbound"
	CertStatusExpired CertStatusType = "CertExpired"
)

type StorageClassInheritDirectiveType string

const (
	StorageClassIDDestinationBucket StorageClassInheritDirectiveType = "DESTINATION_BUCKET"
	StorageClassIDSourceObject      StorageClassInheritDirectiveType = "SOURCE_OBJECT"
)

type StatusType string

const (
	StatusEnabled  StatusType = "Enabled"
	StatusDisabled StatusType = "Disabled"
)

const (
	LifecycleStatusEnabled  StatusType = "Enabled"
	LifecycleStatusDisabled StatusType = "Disabled"
)

type RedirectType string

const (
	RedirectTypeMirror RedirectType = "Mirror"
	RedirectTypeAsync  RedirectType = "Async"
)

const (
	SSETosAlg = "AES256"
	SSEKMS    = "kms"
)

type VersioningStatusType string

const (
	VersioningStatusEnable    VersioningStatusType = "Enabled"
	VersioningStatusSuspended VersioningStatusType = "Suspended"
)

type ProtocolType string

const (
	ProtocolHttp  ProtocolType = "http"
	ProtocolHttps ProtocolType = "https"
)

type CopyEventType int

const (
	CopyEventCreateMultipartUploadSucceed   CopyEventType = 1
	CopyEventCreateMultipartUploadFailed    CopyEventType = 2
	CopyEventUploadPartCopySuccess          CopyEventType = 3
	CopyEventUploadPartCopyFailed           CopyEventType = 4
	CopyEventUploadPartCopyAborted          CopyEventType = 5
	CopyEventCompleteMultipartUploadSucceed CopyEventType = 6
	CopyEventCompleteMultipartUploadFailed  CopyEventType = 7
)

type TierType string

const (
	TierStandard  TierType = "Standard"
	TierExpedited TierType = "Expedited"
	TierBulk      TierType = "Bulk"
)
