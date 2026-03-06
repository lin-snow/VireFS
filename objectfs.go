package virefs

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3API is the subset of *s3.Client that ObjectFS needs.
// Accepting an interface makes unit-testing trivial (no real S3 required).
type S3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// ObjectFS implements FS backed by an S3-compatible object store.
type ObjectFS struct {
	client     S3API
	bucket     string
	basePrefix string // optional prefix prepended to every key
}

// NewObjectFS creates an ObjectFS targeting the given bucket.
//
// basePrefix is joined before every key (e.g. "uploads/") so that a call to
// Get(ctx, "a/b.txt") becomes GetObject with key "uploads/a/b.txt".
// Pass "" for no prefix.
//
// To use a custom endpoint (MinIO, Ceph, R2, …) configure the *s3.Client:
//
//	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
//	    o.BaseEndpoint = aws.String("https://s3.example.com")
//	    o.UsePathStyle = true
//	})
func NewObjectFS(client S3API, bucket, basePrefix string) *ObjectFS {
	return &ObjectFS{
		client:     client,
		bucket:     bucket,
		basePrefix: basePrefix,
	}
}

// s3Key prepends basePrefix and cleans the key.
func (o *ObjectFS) s3Key(key string) (string, error) {
	cleaned, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	return o.basePrefix + cleaned, nil
}

func (o *ObjectFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	s3k, err := o.s3Key(key)
	if err != nil {
		return nil, &OpError{Op: "Get", Key: key, Err: err}
	}
	out, err := o.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(s3k),
	})
	if err != nil {
		return nil, &OpError{Op: "Get", Key: key, Err: mapS3Error(err)}
	}
	return out.Body, nil
}

func (o *ObjectFS) Put(ctx context.Context, key string, r io.Reader) error {
	s3k, err := o.s3Key(key)
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	_, err = o.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(s3k),
		Body:   r,
	})
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: mapS3Error(err)}
	}
	return nil
}

func (o *ObjectFS) Delete(ctx context.Context, key string) error {
	s3k, err := o.s3Key(key)
	if err != nil {
		return &OpError{Op: "Delete", Key: key, Err: err}
	}
	_, err = o.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(s3k),
	})
	if err != nil {
		return &OpError{Op: "Delete", Key: key, Err: mapS3Error(err)}
	}
	return nil
}

func (o *ObjectFS) List(ctx context.Context, prefix string) (*ListResult, error) {
	cleanedPrefix, err := CleanKey(prefix)
	if err != nil {
		return nil, &OpError{Op: "List", Key: prefix, Err: err}
	}
	s3Prefix := o.basePrefix + cleanedPrefix
	if s3Prefix != "" && s3Prefix[len(s3Prefix)-1] != '/' {
		s3Prefix += "/"
	}

	result := &ListResult{}
	var continuationToken *string
	for {
		out, err := o.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(o.bucket),
			Prefix:            aws.String(s3Prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, &OpError{Op: "List", Key: prefix, Err: mapS3Error(err)}
		}
		for _, obj := range out.Contents {
			k := aws.ToString(obj.Key)
			if len(k) > len(o.basePrefix) {
				k = k[len(o.basePrefix):]
			}
			result.Files = append(result.Files, FileInfo{
				Key:          k,
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		continuationToken = out.NextContinuationToken
	}
	return result, nil
}

func (o *ObjectFS) Stat(ctx context.Context, key string) (*FileInfo, error) {
	s3k, err := o.s3Key(key)
	if err != nil {
		return nil, &OpError{Op: "Stat", Key: key, Err: err}
	}
	out, err := o.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(s3k),
	})
	if err != nil {
		return nil, &OpError{Op: "Stat", Key: key, Err: mapS3Error(err)}
	}
	return &FileInfo{
		Key:          key,
		Size:         aws.ToInt64(out.ContentLength),
		LastModified: aws.ToTime(out.LastModified),
	}, nil
}

// mapS3Error converts common S3 error types to virefs sentinel errors.
func mapS3Error(err error) error {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return ErrNotFound
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return ErrNotFound
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" {
			return ErrNotFound
		}
	}
	return err
}
