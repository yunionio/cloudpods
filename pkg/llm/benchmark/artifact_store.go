package benchmark

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

type ArtifactStoreOptions struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Secure    bool
	Prefix    string
}

type ArtifactStore struct {
	opts    ArtifactStoreOptions
	client  *s3.Client
	initErr error
}

var defaultArtifactStore = NewArtifactStore(ArtifactStoreOptions{})

func ConfigureArtifactStore(opts ArtifactStoreOptions) {
	defaultArtifactStore = NewArtifactStore(opts)
}

func DefaultArtifactStore() *ArtifactStore {
	return defaultArtifactStore
}

func NewArtifactStore(opts ArtifactStoreOptions) *ArtifactStore {
	opts.Endpoint = strings.TrimSpace(opts.Endpoint)
	opts.Prefix = strings.Trim(strings.TrimSpace(opts.Prefix), "/")
	ret := &ArtifactStore{opts: opts}
	if opts.Endpoint == "" {
		return ret
	}
	if opts.AccessKey == "" || opts.SecretKey == "" || opts.Bucket == "" {
		ret.initErr = errors.New("missing MinIO/S3 artifact config")
		return ret
	}
	endpoint := opts.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if opts.Secure {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	ret.client = s3.NewFromConfig(aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, ""),
		BaseEndpoint: aws.String(endpoint),
	}, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return ret
}

func (s *ArtifactStore) Enabled() bool {
	return s != nil && s.opts.Endpoint != ""
}

func (s *ArtifactStore) benchmarkPrefix(projectID, benchmarkID string) string {
	return path.Join(s.opts.Prefix, projectID, benchmarkID) + "/"
}

func (s *ArtifactStore) objectKey(projectID, benchmarkID, local string) string {
	return path.Join(s.opts.Prefix, projectID, benchmarkID, filepath.Base(local))
}

func artifactURI(bucket, key string) string {
	return "s3://" + bucket + "/" + key
}

func parseArtifactURI(location string) (string, string, error) {
	parsed, err := url.Parse(location)
	if err != nil || parsed.Scheme != "s3" || parsed.Host == "" {
		return "", "", errors.New("invalid artifact S3 URI")
	}
	key := strings.TrimPrefix(parsed.EscapedPath(), "/")
	key, err = url.PathUnescape(key)
	if err != nil || key == "" {
		return "", "", errors.New("invalid artifact S3 object key")
	}
	return parsed.Host, key, nil
}

func cloneArtifactPaths(paths map[string]string) map[string]string {
	ret := make(map[string]string, len(paths))
	for kind, location := range paths {
		if location != "" {
			ret[kind] = location
		}
	}
	return ret
}

func artifactNotFound(err error) bool {
	var responseError *smithyhttp.ResponseError
	return errors.As(err, &responseError) && responseError.HTTPStatusCode() == http.StatusNotFound
}

func (s *ArtifactStore) Persist(
	ctx context.Context,
	projectID, benchmarkID string,
	files map[string]string,
) (map[string]string, string, error) {
	local := cloneArtifactPaths(files)
	if len(local) == 0 {
		return local, "", nil
	}
	if !s.Enabled() {
		return local, api.LLMBenchmarkArtifactStorageLocal, nil
	}
	if s.initErr != nil {
		return local, api.LLMBenchmarkArtifactStorageLocal, s.initErr
	}
	if err := ensureArtifactBucket(ctx, s.client, s.opts.Bucket); err != nil {
		return local, api.LLMBenchmarkArtifactStorageLocal, err
	}

	kinds := make([]string, 0, len(local))
	for kind := range local {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	remote := make(map[string]string, len(local))
	uploaded := make([]string, 0, len(local))
	for _, kind := range kinds {
		file, err := os.Open(local[kind])
		if err == nil {
			key := s.objectKey(projectID, benchmarkID, local[kind])
			uploaded = append(uploaded, key)
			_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(s.opts.Bucket),
				Key:    aws.String(key),
				Body:   file,
			})
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				remote[kind] = artifactURI(s.opts.Bucket, key)
			}
		}
		if err != nil {
			for _, key := range uploaded {
				_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(s.opts.Bucket),
					Key:    aws.String(key),
				})
			}
			return local, api.LLMBenchmarkArtifactStorageLocal, err
		}
	}
	return remote, api.LLMBenchmarkArtifactStorageMinio, nil
}

func ensureArtifactBucket(ctx context.Context, client *s3.Client, bucket string) error {
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err == nil {
		return nil
	}
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}

func (s *ArtifactStore) RemoveLocal(files map[string]string) error {
	var firstErr error
	for _, file := range files {
		if file == "" {
			continue
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *ArtifactStore) Exists(ctx context.Context, location string) (bool, error) {
	if location == "" {
		return false, nil
	}
	if !strings.HasPrefix(location, "s3://") {
		_, err := os.Stat(location)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}
	if s == nil || s.client == nil {
		if s != nil && s.initErr != nil {
			return false, s.initErr
		}
		return false, errors.New("MinIO/S3 artifact store is disabled")
	}
	bucket, key, err := parseArtifactURI(location)
	if err != nil {
		return false, err
	}
	_, err = s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if artifactNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *ArtifactStore) Open(ctx context.Context, location string) (io.ReadCloser, error) {
	if !strings.HasPrefix(location, "s3://") {
		return os.Open(location)
	}
	if s == nil || s.client == nil {
		if s != nil && s.initErr != nil {
			return nil, s.initErr
		}
		return nil, errors.New("MinIO/S3 artifact store is disabled")
	}
	bucket, key, err := parseArtifactURI(location)
	if err != nil {
		return nil, err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *ArtifactStore) DeleteBenchmark(ctx context.Context, projectID, benchmarkID string) error {
	if !s.Enabled() {
		return nil
	}
	if s.initErr != nil {
		return s.initErr
	}
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.opts.Bucket),
		Prefix: aws.String(s.benchmarkPrefix(projectID, benchmarkID)),
	}
	for {
		out, err := s.client.ListObjectsV2(ctx, input)
		if artifactNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, object := range out.Contents {
			if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.opts.Bucket),
				Key:    object.Key,
			}); err != nil {
				return err
			}
		}
		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			return nil
		}
		input.ContinuationToken = out.NextContinuationToken
	}
}
