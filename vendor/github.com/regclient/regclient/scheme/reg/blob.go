package reg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/reghttp"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/blob"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
)

// BlobDelete removes a blob from the repository
func (reg *Reg) BlobDelete(ctx context.Context, r ref.Ref, d types.Descriptor) error {
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "DELETE",
				Repository: r.Repository,
				Path:       "blobs/" + d.Digest.String(),
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete blob, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), err)
	}
	if resp.HTTPResponse().StatusCode != 202 {
		return fmt.Errorf("failed to delete blob, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	return nil
}

// BlobGet retrieves a blob from the repository, returning a blob reader
func (reg *Reg) BlobGet(ctx context.Context, r ref.Ref, d types.Descriptor) (blob.Reader, error) {
	// build/send request
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "GET",
				Repository: r.Repository,
				Path:       "blobs/" + d.Digest.String(),
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil && len(d.URLs) > 0 {
		for _, curURL := range d.URLs {
			// fallback for external blobs
			var u *url.URL
			u, err = url.Parse(curURL)
			if err != nil {
				return nil, fmt.Errorf("failed to parse external url \"%s\": %w", curURL, err)
			}
			req = &reghttp.Req{
				Host: r.Registry,
				APIs: map[string]reghttp.ReqAPI{
					"": {
						Method:     "GET",
						Repository: r.Repository,
						DirectURL:  u,
					},
				},
				NoMirrors: true,
			}
			resp, err = reg.reghttp.Do(ctx, req)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get blob, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), err)
	}
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to get blob, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	b := blob.NewReader(
		blob.WithRef(r),
		blob.WithReader(resp),
		blob.WithDesc(types.Descriptor{
			Digest: d.Digest,
		}),
		blob.WithResp(resp.HTTPResponse()),
	)
	return b, nil
}

// BlobHead is used to verify if a blob exists and is accessible
func (reg *Reg) BlobHead(ctx context.Context, r ref.Ref, d types.Descriptor) (blob.Reader, error) {
	// build/send request
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "HEAD",
				Repository: r.Repository,
				Path:       "blobs/" + d.Digest.String(),
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil && len(d.URLs) > 0 {
		for _, curURL := range d.URLs {
			// fallback for external blobs
			var u *url.URL
			u, err = url.Parse(curURL)
			if err != nil {
				return nil, fmt.Errorf("failed to parse external url \"%s\": %w", curURL, err)
			}
			req = &reghttp.Req{
				Host: r.Registry,
				APIs: map[string]reghttp.ReqAPI{
					"": {
						Method:     "HEAD",
						Repository: r.Repository,
						DirectURL:  u,
					},
				},
				NoMirrors: true,
			}
			resp, err = reg.reghttp.Do(ctx, req)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to request blob head, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to request blob head, digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	b := blob.NewReader(
		blob.WithRef(r),
		blob.WithDesc(types.Descriptor{
			Digest: d.Digest,
		}),
		blob.WithResp(resp.HTTPResponse()),
	)
	return b, nil
}

// BlobMount attempts to perform a server side copy/mount of the blob between repositories
func (reg *Reg) BlobMount(ctx context.Context, rSrc ref.Ref, rTgt ref.Ref, d types.Descriptor) error {
	_, uuid, err := reg.blobMount(ctx, rTgt, d, rSrc)
	// if mount fails and returns an upload location, cancel that upload
	if err != nil {
		reg.blobUploadCancel(ctx, rTgt, uuid)
	}
	return err
}

// BlobPut uploads a blob to a repository.
// This will attempt an anonymous blob mount first which some registries may support.
// It will then try doing a full put of the blob without chunking (most widely supported).
// If the full put fails, it will fall back to a chunked upload (useful for flaky networks).
func (reg *Reg) BlobPut(ctx context.Context, r ref.Ref, d types.Descriptor, rdr io.Reader) (types.Descriptor, error) {
	var putURL *url.URL
	var err error
	// defaults for content-type and length
	if d.Size == 0 {
		d.Size = -1
	}

	// attempt an anonymous blob mount
	if d.Digest != "" && d.Size > 0 {
		putURL, _, err = reg.blobMount(ctx, r, d, ref.Ref{})
		if err == nil {
			return d, nil
		}
		if err != types.ErrMountReturnedLocation {
			putURL = nil
		}
	}
	// fallback to requesting upload URL
	if putURL == nil {
		putURL, err = reg.blobGetUploadURL(ctx, r)
		if err != nil {
			return d, err
		}
	}

	// send upload as one-chunk
	tryPut := bool(d.Digest != "" && d.Size > 0)
	if tryPut {
		host := reg.hostGet(r.Registry)
		maxPut := host.BlobMax
		if maxPut == 0 {
			maxPut = reg.blobMaxPut
		}
		if maxPut > 0 && d.Size > maxPut {
			tryPut = false
		}
	}
	if tryPut {
		err = reg.blobPutUploadFull(ctx, r, d, putURL, rdr)
		if err == nil {
			return d, nil
		}
		// on failure, attempt to seek back to start to perform a chunked upload
		rdrSeek, ok := rdr.(io.ReadSeeker)
		if !ok {
			return d, err
		}
		offset, errR := rdrSeek.Seek(0, io.SeekStart)
		if errR != nil || offset != 0 {
			return d, err
		}
	}

	// send a chunked upload if full upload not possible or too large
	return reg.blobPutUploadChunked(ctx, r, putURL, rdr)
}

func (reg *Reg) blobGetUploadURL(ctx context.Context, r ref.Ref) (*url.URL, error) {
	// request an upload location
	req := &reghttp.Req{
		Host:      r.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "POST",
				Repository: r.Repository,
				Path:       "blobs/uploads/",
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send blob post, ref %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 202 {
		return nil, fmt.Errorf("failed to send blob post, ref %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	// if min size header received, check/adjust host settings
	minSizeStr := resp.HTTPResponse().Header.Get(blobChunkMinHeader)
	if minSizeStr != "" {
		minSize, err := strconv.ParseInt(minSizeStr, 10, 64)
		if err != nil {
			reg.log.WithFields(logrus.Fields{
				"size": minSizeStr,
				"err":  err,
			}).Warn("Failed to parse chunk size header")
		} else {
			host := reg.hostGet(r.Registry)
			if (host.BlobChunk > 0 && minSize > host.BlobChunk) || (host.BlobChunk <= 0 && minSize > reg.blobChunkSize) {
				if minSize > reg.blobChunkLimit {
					host.BlobChunk = reg.blobChunkLimit
				} else {
					host.BlobChunk = minSize
				}
				reg.log.WithFields(logrus.Fields{
					"size": host.BlobChunk,
					"host": host.Name,
				}).Debug("Registry requested min chunk size")
			}
		}
	}
	// Extract the location into a new putURL based on whether it's relative, fqdn with a scheme, or without a scheme.
	location := resp.HTTPResponse().Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("failed to send blob post, ref %s: %w", r.CommonName(), types.ErrMissingLocation)
	}
	reg.log.WithFields(logrus.Fields{
		"location": location,
	}).Debug("Upload location received")
	// put url may be relative to the above post URL, so parse in that context
	postURL := resp.HTTPResponse().Request.URL
	putURL, err := postURL.Parse(location)
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"location": location,
			"err":      err,
		}).Warn("Location url failed to parse")
		return nil, fmt.Errorf("blob upload url invalid, ref %s: %w", r.CommonName(), err)
	}
	return putURL, nil
}

func (reg *Reg) blobMount(ctx context.Context, rTgt ref.Ref, d types.Descriptor, rSrc ref.Ref) (*url.URL, string, error) {
	// build/send request
	query := url.Values{}
	query.Set("mount", d.Digest.String())
	ignoreErr := true // ignore errors from anonymous blob mount attempts
	if rSrc.Registry == rTgt.Registry && rSrc.Repository != "" {
		query.Set("from", rSrc.Repository)
		ignoreErr = false
	}

	req := &reghttp.Req{
		Host:      rTgt.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "POST",
				Repository: rTgt.Repository,
				Path:       "blobs/uploads/",
				Query:      query,
				IgnoreErr:  ignoreErr,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to mount blob, digest %s, ref %s: %w", d.Digest.String(), rTgt.CommonName(), err)
	}
	defer resp.Close()

	// if min size header received, check/adjust host settings
	minSizeStr := resp.HTTPResponse().Header.Get(blobChunkMinHeader)
	if minSizeStr != "" {
		minSize, err := strconv.ParseInt(minSizeStr, 10, 64)
		if err != nil {
			reg.log.WithFields(logrus.Fields{
				"size": minSizeStr,
				"err":  err,
			}).Warn("Failed to parse chunk size header")
		} else {
			host := reg.hostGet(rTgt.Registry)
			if (host.BlobChunk > 0 && minSize > host.BlobChunk) || (host.BlobChunk <= 0 && minSize > reg.blobChunkSize) {
				if minSize > reg.blobChunkLimit {
					host.BlobChunk = reg.blobChunkLimit
				} else {
					host.BlobChunk = minSize
				}
				reg.log.WithFields(logrus.Fields{
					"size": host.BlobChunk,
					"host": host.Name,
				}).Debug("Registry requested min chunk size")
			}
		}
	}
	// 201 indicates the blob mount succeeded
	if resp.HTTPResponse().StatusCode == 201 {
		return nil, "", nil
	}
	// 202 indicates blob mount failed but server ready to receive an upload at location
	location := resp.HTTPResponse().Header.Get("Location")
	uuid := resp.HTTPResponse().Header.Get("Docker-Upload-UUID")
	if resp.HTTPResponse().StatusCode == 202 && location != "" {
		postURL := resp.HTTPResponse().Request.URL
		putURL, err := postURL.Parse(location)
		if err != nil {
			reg.log.WithFields(logrus.Fields{
				"digest":   d,
				"target":   rTgt.CommonName(),
				"location": location,
				"err":      err,
			}).Warn("Mount location header failed to parse")
		} else {
			return putURL, uuid, types.ErrMountReturnedLocation
		}
	}
	// all other responses unhandled
	return nil, "", fmt.Errorf("failed to mount blob, digest %s, ref %s: %w", d.Digest.String(), rTgt.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
}

func (reg *Reg) blobPutUploadFull(ctx context.Context, r ref.Ref, d types.Descriptor, putURL *url.URL, rdr io.Reader) error {
	// append digest to request to use the monolithic upload option
	if putURL.RawQuery != "" {
		putURL.RawQuery = putURL.RawQuery + "&digest=" + url.QueryEscape(d.Digest.String())
	} else {
		putURL.RawQuery = "digest=" + url.QueryEscape(d.Digest.String())
	}

	// make a reader function for the blob
	readOnce := false
	bodyFunc := func() (io.ReadCloser, error) {
		// if reader is reused,
		if readOnce {
			rdrSeek, ok := rdr.(io.ReadSeeker)
			if !ok {
				return nil, fmt.Errorf("unable to reuse reader")
			}
			_, err := rdrSeek.Seek(0, io.SeekStart)
			if err != nil {
				return nil, fmt.Errorf("unable to reuse reader")
			}
		}
		readOnce = true
		return io.NopCloser(rdr), nil
	}

	// build/send request
	header := http.Header{
		"Content-Type": {"application/octet-stream"},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "PUT",
				Repository: r.Repository,
				DirectURL:  putURL,
				BodyFunc:   bodyFunc,
				BodyLen:    d.Size,
				Headers:    header,
			},
		},
		NoMirrors: true,
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send blob (put), digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), err)
	}
	defer resp.Close()
	// 201 follows distribution-spec, 204 is listed as possible in the Docker registry spec
	if resp.HTTPResponse().StatusCode != 201 && resp.HTTPResponse().StatusCode != 204 {
		return fmt.Errorf("failed to send blob (put), digest %s, ref %s: %w", d.Digest.String(), r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	return nil
}

func (reg *Reg) blobPutUploadChunked(ctx context.Context, r ref.Ref, putURL *url.URL, rdr io.Reader) (types.Descriptor, error) {
	host := reg.hostGet(r.Registry)
	bufSize := host.BlobChunk
	if bufSize <= 0 {
		bufSize = reg.blobChunkSize
	}
	bufBytes := make([]byte, 0, bufSize)
	bufRdr := bytes.NewReader(bufBytes)
	bufStart := int64(0)
	bufChange := false

	// setup buffer and digest pipe
	digester := digest.Canonical.Digester()
	digestRdr := io.TeeReader(rdr, digester.Hash())
	finalChunk := false
	chunkStart := int64(0)
	chunkSize := 0
	bodyFunc := func() (io.ReadCloser, error) {
		// reset to the start on every new read
		_, err := bufRdr.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bufRdr), nil
	}
	chunkURL := *putURL
	retryLimit := 10 // TODO: pull limit from reghttp
	retryCur := 0
	var err error

	for !finalChunk || chunkStart < bufStart+int64(len(bufBytes)) {
		bufChange = false
		for chunkStart >= bufStart+int64(len(bufBytes)) && !finalChunk {
			bufStart += int64(len(bufBytes))
			// reset length if previous read was short
			if cap(bufBytes) != len(bufBytes) {
				bufBytes = bufBytes[:cap(bufBytes)]
				bufChange = true
			}
			// read a chunk into an input buffer, computing the digest
			chunkSize, err = io.ReadFull(digestRdr, bufBytes)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				finalChunk = true
			} else if err != nil {
				return types.Descriptor{}, fmt.Errorf("failed to send blob chunk, ref %s: %w", r.CommonName(), err)
			}
			// update length on partial read
			if chunkSize != len(bufBytes) {
				bufBytes = bufBytes[:chunkSize]
				bufChange = true
			}
		}
		if chunkStart > bufStart && chunkStart < bufStart+int64(len(bufBytes)) {
			// next chunk is inside the existing buf
			bufBytes = bufBytes[chunkStart-bufStart:]
			bufStart = chunkStart
			chunkSize = len(bufBytes)
			bufChange = true
		}
		if chunkSize > 0 && chunkStart != bufStart {
			return types.Descriptor{}, fmt.Errorf("chunkStart (%d) != bufStart (%d)", chunkStart, bufStart)
		}
		if bufChange {
			// need to recreate the reader on a change to the slice length,
			// old reader is looking at the old slice metadata
			bufRdr = bytes.NewReader(bufBytes)
		}

		if chunkSize > 0 {
			// write chunk
			header := http.Header{
				"Content-Type":  {"application/octet-stream"},
				"Content-Range": {fmt.Sprintf("%d-%d", chunkStart, chunkStart+int64(chunkSize)-1)},
			}
			req := &reghttp.Req{
				Host: r.Registry,
				APIs: map[string]reghttp.ReqAPI{
					"": {
						Method:     "PATCH",
						Repository: r.Repository,
						DirectURL:  &chunkURL,
						BodyFunc:   bodyFunc,
						BodyLen:    int64(chunkSize),
						Headers:    header,
					},
				},
				NoMirrors: true,
			}
			resp, err := reg.reghttp.Do(ctx, req)
			if err != nil && !errors.Is(err, types.ErrHTTPStatus) && !errors.Is(err, types.ErrNotFound) {
				return types.Descriptor{}, fmt.Errorf("failed to send blob (chunk), ref %s: http do: %w", r.CommonName(), err)
			}
			resp.Close()
			httpResp := resp.HTTPResponse()
			// distribution-spec is 202, AWS ECR returns a 201 and rejects the put
			if resp.HTTPResponse().StatusCode == 201 {
				reg.log.WithFields(logrus.Fields{
					"ref":        r.CommonName(),
					"chunkStart": chunkStart,
					"chunkSize":  chunkSize,
				}).Debug("Early accept of chunk in PATCH before PUT request")
			} else if resp.HTTPResponse().StatusCode >= 400 && resp.HTTPResponse().StatusCode < 500 &&
				resp.HTTPResponse().Header.Get("Location") != "" &&
				resp.HTTPResponse().Header.Get("Range") != "" {
				retryCur++
				reg.log.WithFields(logrus.Fields{
					"ref":        r.CommonName(),
					"chunkStart": chunkStart,
					"chunkSize":  chunkSize,
					"range":      resp.HTTPResponse().Header.Get("Range"),
				}).Debug("Recoverable chunk upload error")
			} else if resp.HTTPResponse().StatusCode != 202 {
				retryCur++
				statusResp, statusErr := reg.blobUploadStatus(ctx, r, &chunkURL)
				if retryCur > retryLimit || statusErr != nil {
					return types.Descriptor{}, fmt.Errorf("failed to send blob (chunk), ref %s: http status: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
				}
				httpResp = statusResp
			} else {
				// successful request
				if retryCur > 0 {
					retryCur--
				}
			}
			rangeEnd, err := blobUploadCurBytes(httpResp)
			if err == nil {
				chunkStart = rangeEnd + 1
			} else {
				chunkStart += int64(chunkSize)
			}
			location := httpResp.Header.Get("Location")
			if location != "" {
				reg.log.WithFields(logrus.Fields{
					"location": location,
				}).Debug("Next chunk upload location received")
				prevURL := httpResp.Request.URL
				parseURL, err := prevURL.Parse(location)
				if err != nil {
					return types.Descriptor{}, fmt.Errorf("failed to send blob (parse next chunk location), ref %s: %w", r.CommonName(), err)
				}
				chunkURL = *parseURL
			}
		}
	}

	// compute digest
	d := digester.Digest()

	// send the final put
	// append digest to request to use the monolithic upload option
	if chunkURL.RawQuery != "" {
		chunkURL.RawQuery = chunkURL.RawQuery + "&digest=" + url.QueryEscape(d.String())
	} else {
		chunkURL.RawQuery = "digest=" + url.QueryEscape(d.String())
	}

	header := http.Header{
		"Content-Type": {"application/octet-stream"},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "PUT",
				Repository: r.Repository,
				DirectURL:  &chunkURL,
				BodyLen:    int64(0),
				Headers:    header,
			},
		},
		NoMirrors: true,
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return types.Descriptor{}, fmt.Errorf("failed to send blob (chunk digest), digest %s, ref %s: %w", d, r.CommonName(), err)
	}
	defer resp.Close()
	// 201 follows distribution-spec, 204 is listed as possible in the Docker registry spec
	if resp.HTTPResponse().StatusCode != 201 && resp.HTTPResponse().StatusCode != 204 {
		return types.Descriptor{}, fmt.Errorf("failed to send blob (chunk digest), digest %s, ref %s: %w", d, r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	return types.Descriptor{Digest: d, Size: chunkStart}, nil
}

// TODO: just take a putURL rather than the uuid and call a delete on that url
func (reg *Reg) blobUploadCancel(ctx context.Context, r ref.Ref, uuid string) error {
	if uuid == "" {
		return fmt.Errorf("failed to cancel upload %s: uuid undefined", r.CommonName())
	}
	req := &reghttp.Req{
		Host:      r.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "DELETE",
				Repository: r.Repository,
				Path:       "blobs/uploads/" + uuid,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to cancel upload %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 202 {
		return fmt.Errorf("failed to cancel upload %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	return nil
}

// blobUploadStatus provides a response with headers indicating the progress of an upload
func (reg *Reg) blobUploadStatus(ctx context.Context, r ref.Ref, putURL *url.URL) (*http.Response, error) {
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "GET",
				Repository: r.Repository,
				DirectURL:  putURL,
			},
		},
		NoMirrors: true,
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload status: %v", err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 204 {
		return resp.HTTPResponse(), fmt.Errorf("failed to get upload status: %w", reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	return resp.HTTPResponse(), nil
}

func blobUploadCurBytes(resp *http.Response) (int64, error) {
	if resp == nil {
		return 0, fmt.Errorf("missing response")
	}
	r := resp.Header.Get("Range")
	if r == "" {
		return 0, fmt.Errorf("missing range header")
	}
	rSplit := strings.SplitN(r, "-", 2)
	if len(rSplit) < 2 {
		return 0, fmt.Errorf("missing offset in range header")
	}
	return strconv.ParseInt(rSplit[1], 10, 64)
}
