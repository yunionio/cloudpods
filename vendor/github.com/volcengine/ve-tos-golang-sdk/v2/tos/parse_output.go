package tos

import (
	"net/http"
	"strconv"
)

// ParseListObjectsType2Output Parse the incoming parameters of *http.Response type, and respond to the return value of *ListObjectsType2Output type.
func ParseListObjectsType2Output(httpRes *http.Response) (*ListObjectsType2Output, error) {

	res := &Response{
		StatusCode:    httpRes.StatusCode,
		ContentLength: httpRes.ContentLength,
		Header:        httpRes.Header,
		Body:          httpRes.Body,
	}

	err := checkError(res, true, 200)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	temp := listObjectsType2Output{
		RequestInfo: res.RequestInfo(),
	}
	if err = marshalOutput(temp.RequestID, res.Body, &temp); err != nil {
		return nil, err
	}
	contents := make([]ListedObjectV2, 0, len(temp.Contents))
	for _, object := range temp.Contents {
		var hashCrc uint64
		if len(object.HashCrc64ecma) == 0 {
			hashCrc = 0
		} else {
			hashCrc, err = strconv.ParseUint(object.HashCrc64ecma, 10, 64)
			if err != nil {
				return nil, &TosServerError{
					TosError:    TosError{Message: "tos: server returned invalid HashCrc64Ecma"},
					RequestInfo: RequestInfo{RequestID: temp.RequestID},
				}
			}
		}
		contents = append(contents, ListedObjectV2{
			Key:           object.Key,
			LastModified:  object.LastModified,
			ETag:          object.ETag,
			Size:          object.Size,
			Owner:         object.Owner,
			StorageClass:  object.StorageClass,
			HashCrc64ecma: hashCrc,
		})
	}
	output := ListObjectsType2Output{
		RequestInfo:           temp.RequestInfo,
		Name:                  temp.Name,
		ContinuationToken:     temp.ContinuationToken,
		Prefix:                temp.Prefix,
		MaxKeys:               temp.MaxKeys,
		KeyCount:              temp.KeyCount,
		Delimiter:             temp.Delimiter,
		IsTruncated:           temp.IsTruncated,
		EncodingType:          temp.EncodingType,
		CommonPrefixes:        temp.CommonPrefixes,
		NextContinuationToken: temp.NextContinuationToken,
		Contents:              contents,
	}
	return &output, nil
}

// ParseListObjectsV2Output Parse the incoming parameters of *http.Response type, and respond to the return value of *ListObjectsV2Output type.
func ParseListObjectsV2Output(httpRes *http.Response) (*ListObjectsV2Output, error) {

	res := &Response{
		StatusCode:    httpRes.StatusCode,
		ContentLength: httpRes.ContentLength,
		Header:        httpRes.Header,
		Body:          httpRes.Body,
	}

	err := checkError(res, true, 200)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	temp := listObjectsV2Output{
		RequestInfo: res.RequestInfo(),
	}
	if err = marshalOutput(temp.RequestID, res.Body, &temp); err != nil {
		return nil, err
	}
	contents := make([]ListedObjectV2, 0, len(temp.Contents))
	for _, object := range temp.Contents {
		var hashCrc uint64
		if len(object.HashCrc64ecma) == 0 {
			hashCrc = 0
		} else {
			hashCrc, err = strconv.ParseUint(object.HashCrc64ecma, 10, 64)
			if err != nil {
				return nil, &TosServerError{
					TosError:    TosError{Message: "tos: server returned invalid HashCrc64Ecma"},
					RequestInfo: RequestInfo{RequestID: temp.RequestID},
				}
			}
		}
		contents = append(contents, ListedObjectV2{
			Key:           object.Key,
			LastModified:  object.LastModified,
			ETag:          object.ETag,
			Size:          object.Size,
			Owner:         object.Owner,
			StorageClass:  object.StorageClass,
			HashCrc64ecma: uint64(hashCrc),
		})
	}
	output := ListObjectsV2Output{
		RequestInfo:    temp.RequestInfo,
		Name:           temp.Name,
		Prefix:         temp.Prefix,
		Marker:         temp.Marker,
		MaxKeys:        temp.MaxKeys,
		NextMarker:     temp.NextMarker,
		Delimiter:      temp.Delimiter,
		IsTruncated:    temp.IsTruncated,
		EncodingType:   temp.EncodingType,
		CommonPrefixes: temp.CommonPrefixes,
		Contents:       contents,
	}
	return &output, nil
}

// ParseListObjectVersionsV2Output Parse the incoming parameters of *http.Response type, and respond to the return value of *ListObjectVersionsV2Output type.
func ParseListObjectVersionsV2Output(httpRes *http.Response) (*ListObjectVersionsV2Output, error) {

	res := &Response{
		StatusCode:    httpRes.StatusCode,
		ContentLength: httpRes.ContentLength,
		Header:        httpRes.Header,
		Body:          httpRes.Body,
	}

	err := checkError(res, true, 200)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	temp := listObjectVersionsV2Output{RequestInfo: res.RequestInfo()}

	if err = marshalOutput(temp.RequestID, res.Body, &temp); err != nil {
		return nil, err
	}
	versions := make([]ListedObjectVersionV2, 0, len(temp.Versions))
	for _, version := range temp.Versions {
		var hashCrc uint64
		if len(version.HashCrc64ecma) == 0 {
			hashCrc = 0
		} else {
			hashCrc, err = strconv.ParseUint(version.HashCrc64ecma, 10, 64)
			if err != nil {
				return nil, &TosServerError{
					TosError:    TosError{Message: "tos: server returned invalid HashCrc64Ecma"},
					RequestInfo: RequestInfo{RequestID: temp.RequestID},
				}
			}
		}
		versions = append(versions, ListedObjectVersionV2{
			Key:           version.Key,
			LastModified:  version.LastModified,
			ETag:          version.ETag,
			IsLatest:      version.IsLatest,
			Size:          version.Size,
			Owner:         version.Owner,
			StorageClass:  version.StorageClass,
			VersionID:     version.VersionID,
			HashCrc64ecma: hashCrc,
		})
	}
	output := ListObjectVersionsV2Output{
		RequestInfo:         temp.RequestInfo,
		Name:                temp.Name,
		Prefix:              temp.Prefix,
		KeyMarker:           temp.KeyMarker,
		VersionIDMarker:     temp.VersionIDMarker,
		Delimiter:           temp.Delimiter,
		EncodingType:        temp.EncodingType,
		MaxKeys:             temp.MaxKeys,
		NextKeyMarker:       temp.NextKeyMarker,
		NextVersionIDMarker: temp.NextVersionIDMarker,
		IsTruncated:         temp.IsTruncated,
		CommonPrefixes:      temp.CommonPrefixes,
		DeleteMarkers:       temp.DeleteMarkers,
		Versions:            versions,
	}

	return &output, nil
}

// ParseHeadObjectV2Output Parse the incoming parameters of *http.Response type, and respond to the return value of *HeadObjectV2Output type.
func ParseHeadObjectV2Output(httpRes *http.Response) (*HeadObjectV2Output, error) {

	res := &Response{
		StatusCode:    httpRes.StatusCode,
		ContentLength: httpRes.ContentLength,
		Header:        httpRes.Header,
		Body:          httpRes.Body,
	}

	err := checkError(res, false, 200)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := HeadObjectV2Output{
		RequestInfo: res.RequestInfo(),
	}
	output.ObjectMetaV2.fromResponseV2(res)
	return &output, nil
}

// ParseGetObjectV2Output Parse the incoming parameters of *http.Response type, and respond to the return value of *GetObjectV2Output type.
func ParseGetObjectV2Output(httpRes *http.Response, expectedCode int) (*GetObjectV2Output, error) {

	res := &Response{
		StatusCode:    httpRes.StatusCode,
		ContentLength: httpRes.ContentLength,
		Header:        httpRes.Header,
		Body:          httpRes.Body,
	}

	err := checkError(res, true, expectedCode)
	if err != nil {
		return nil, err
	}

	basic := GetObjectBasicOutput{
		RequestInfo:  res.RequestInfo(),
		ContentRange: res.Header.Get(HeaderContentRange),
	}
	basic.ObjectMetaV2.fromResponseV2(res)

	output := GetObjectV2Output{
		GetObjectBasicOutput: basic,
		Content:              wrapReader(res.Body, res.ContentLength, nil, nil, nil),
	}
	return &output, nil
}
