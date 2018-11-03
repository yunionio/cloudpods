package cos

import (
	"fmt"
)

// ListAllMyBucketsResult 获取bucket列表的结果
type ListAllMyBucketsResult struct {
	Owner struct {
		ID          string
		DisplayName string
	}

	Buckets struct {
		Bucket []struct {
			Name       string
			Location   string
			CreateDate string
		}
	}
}

// Error 错误消息
type Error struct {
	Code      string
	Message   string
	Resource  string
	RequestID string `xml:"RequestId"`
	TraceID   string `xml:"TaceId"`
}

// HTTPError http error struct
type HTTPError struct {
	Code    int
	Message string
}

// Error  error interface
func (he HTTPError) Error() string {
	return fmt.Sprintf("%d:%s", he.Code, he.Message)
}

// AccessControlPolicy acl return
type AccessControlPolicy struct {
	Owner struct {
		ID          string
		DisplayName string
	}
	AccessControlList struct {
		Grant []struct {
			Grantee struct {
				ID          string
				DisplayName string
			}
			Permission string
		}
	}
}

// ListBucketResult list bucket contents result
type ListBucketResult struct {
	Name         string
	EncodingType string `xml:"Encoding-Type"`
	Prefix       string
	Marker       string
	MaxKeys      int
	IsTruncated  bool
	NextMarker   string
	Contents     []struct {
		Key          string
		LastModified string
		ETag         string
		Size         int64
		Owner        struct {
			ID string
		}
		StorageClass string
	}
	CommonPrefixes []struct {
		Prefix string
	}
}

// ListMultipartUploadsResult list uploading task
type ListMultipartUploadsResult struct {
	Bucket             string
	EncodingType       string `xml:"Encoding-Type"`
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	MaxUploads         int
	IsTruncated        bool
	Prefix             string
	Delimiter          string
	Upload             []struct {
		Key          string
		UploadID     string
		StorageClass string
		Initiator    struct {
			UIN string
		}
		Owner struct {
			UID string
		}
		Initiated string
	}
	CommonPrefixes []struct {
		Prefix string
	}
}

// InitiateMultipartUploadResult init slice upload
type InitiateMultipartUploadResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`
}

// CompleteMultipartUploadResult compeleted slice upload
type CompleteMultipartUploadResult struct {
	Location string
	Bucket   string
	Key      string
	ETag     string
}

// SliceError slice upload err
type SliceError struct {
	Message string
}

// Error implements error
func (se SliceError) Error() string {
	return fmt.Sprintf("上传分片失败:%s", se.Message)
}

// ParamError slice upload err
type ParamError struct {
	Message string
}

// Error implements error
func (pe ParamError) Error() string {
	return fmt.Sprintf("参数错误:%s", pe.Message)
}

// FileError slice upload err
type FileError struct {
	Message string
}

// Error implements error
func (fe FileError) Error() string {
	return fmt.Sprintf("文件错误:%s", fe.Message)
}
