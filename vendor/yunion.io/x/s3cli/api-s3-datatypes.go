/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package s3cli

import (
	"encoding/xml"
	"time"
)

// listAllMyBucketsResult container for listBuckets response.
type ListAllMyBucketsResult struct {
	// Container for one or more buckets.
	Buckets struct {
		Bucket []BucketInfo
	}
	Owner Owner
}

// owner container for bucket owner information.
type Owner struct {
	DisplayName string
	ID          string
}

// CommonPrefix container for prefix response.
type CommonPrefix struct {
	Prefix string
}

// ListBucketV2Result container for listObjects response version 2.
type ListBucketV2Result struct {
	// A response can contain CommonPrefixes only if you have
	// specified a delimiter.
	CommonPrefixes []CommonPrefix
	// Metadata about each object returned.
	Contents  []ObjectInfo
	Delimiter string

	// Encoding type used to encode object keys in the response.
	EncodingType string

	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	MaxKeys     int64
	Name        string

	// Hold the token that will be sent in the next request to fetch the next group of keys
	NextContinuationToken string

	ContinuationToken string
	Prefix            string

	// FetchOwner and StartAfter are currently not used
	FetchOwner string
	StartAfter string
}

// ListBucketResult container for listObjects response.
type ListBucketResult struct {
	// A response can contain CommonPrefixes only if you have
	// specified a delimiter.
	CommonPrefixes []CommonPrefix
	// Metadata about each object returned.
	Contents  []ObjectInfo
	Delimiter string

	// Encoding type used to encode object keys in the response.
	EncodingType string

	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	Marker      string
	MaxKeys     int64
	Name        string

	// When response is truncated (the IsTruncated element value in
	// the response is true), you can use the key name in this field
	// as marker in the subsequent request to get next set of objects.
	// Object storage lists objects in alphabetical order Note: This
	// element is returned only if you have delimiter request
	// parameter specified. If response does not include the NextMaker
	// and it is truncated, you can use the value of the last Key in
	// the response as the marker in the subsequent request to get the
	// next set of object keys.
	NextMarker string
	Prefix     string
}

type ListMultipartUploadsInput struct {
	Delimiter      string
	MaxUploads     int64
	KeyMarker      string
	Prefix         string
	UploadIdMarker string
	EncodingType   string
}

// ListMultipartUploadsResult container for ListMultipartUploads response
type ListMultipartUploadsResult struct {
	Bucket             string
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	EncodingType       string
	MaxUploads         int64
	IsTruncated        bool
	Uploads            []ObjectMultipartInfo `xml:"Upload"`
	Prefix             string
	Delimiter          string
	// A response can contain CommonPrefixes only if you specify a delimiter.
	CommonPrefixes []CommonPrefix
}

// initiator container for who initiated multipart upload.
type initiator struct {
	ID          string
	DisplayName string
}

// copyObjectResult container for copy object response.
type CopyObjectResult struct {
	ETag         string
	LastModified time.Time // time string format "2006-01-02T15:04:05.000Z"
}

// copyPartResult container for copy part response
type CopyPartResult struct {
	ETag         string
	LastModified time.Time // time string format "2006-01-02T15:04:05.000Z"
}

// ObjectPart container for particular part of an object.
type ObjectPart struct {
	// Part number identifies the part.
	PartNumber int

	// Date and time the part was uploaded.
	LastModified time.Time

	// Entity tag returned when the part was uploaded, usually md5sum
	// of the part.
	ETag string

	// Size of the uploaded part data.
	Size int64
}

// ListObjectPartsResult container for ListObjectParts response.
type ListObjectPartsResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`

	Initiator initiator
	Owner     Owner

	StorageClass         string
	PartNumberMarker     int
	NextPartNumberMarker int
	MaxParts             int

	// Indicates whether the returned list of parts is truncated.
	IsTruncated bool
	ObjectParts []ObjectPart `xml:"Part"`

	EncodingType string
}

// initiateMultipartUploadResult container for InitiateMultiPartUpload
// response.
type InitiateMultipartUploadResult struct {
	XMLName xml.Name `xml:"InitiateMultipartUploadResult" json:"-"`

	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	UploadID string `xml:"UploadId"`
}

// completeMultipartUploadResult container for completed multipart
// upload response.
type CompleteMultipartUploadResult struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult" json:"-"`

	Location string `xml:"Location"`
	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	ETag     string `xml:"ETag"`
}

// CompletePart sub container lists individual part numbers and their
// md5sum, part of completeMultipartUpload.
type CompletePart struct {
	XMLName xml.Name `xml:"Part" json:"-"`

	// Part number identifies the part.
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

// completeMultipartUpload container for completing multipart upload.
type CompleteMultipartUpload struct {
	XMLName xml.Name       `xml:"CompleteMultipartUpload" json:"-"`
	Parts   []CompletePart `xml:"Part"`
}

// createBucketConfiguration container for bucket configuration.
type CreateBucketConfiguration struct {
	XMLName  xml.Name `xml:"CreateBucketConfiguration" json:"-"`
	Location string   `xml:"LocationConstraint"`
}

// LocationConstraint
type LocationConstraint string

// deleteObject container for Delete element in MultiObjects Delete XML request
type deleteObject struct {
	Key       string
	VersionID string `xml:"VersionId,omitempty"`
}

// deletedObject container for Deleted element in MultiObjects Delete XML response
type deletedObject struct {
	Key       string
	VersionID string `xml:"VersionId,omitempty"`
	// These fields are ignored.
	DeleteMarker          bool
	DeleteMarkerVersionID string
}

// nonDeletedObject container for Error element (failed deletion) in MultiObjects Delete XML response
type nonDeletedObject struct {
	Key     string
	Code    string
	Message string
}

// deletedMultiObjects container for MultiObjects Delete XML request
type deleteMultiObjects struct {
	XMLName xml.Name `xml:"Delete"`
	Quiet   bool
	Objects []deleteObject `xml:"Object"`
}

// deletedMultiObjectsResult container for MultiObjects Delete XML response
type deleteMultiObjectsResult struct {
	XMLName          xml.Name           `xml:"DeleteResult"`
	DeletedObjects   []deletedObject    `xml:"Deleted"`
	UnDeletedObjects []nonDeletedObject `xml:"Error"`
}

type VersioningConfiguration struct {
	XMLName   xml.Name `xml:"VersioningConfiguration"`
	Status    string   `xml:"Status,omitempty"`
	MfaDelete string   `xml:"MfaDelete,omitempty"`
}
