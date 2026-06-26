package artifacts

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store stores artifacts in an S3-compatible bucket.
// Works with AWS S3, MinIO, Backblaze B2, and any S3-compatible service.
type S3Store struct {
	client   *s3.Client
	bucket   string
	prefix   string
}

// NewS3Store creates an S3Store using the default AWS credential chain.
// endpoint overrides the S3 endpoint URL (pass "" for standard AWS S3).
func NewS3Store(ctx context.Context, bucket, prefix, endpoint string) (*S3Store, error) {
	opts := []func(*config.LoadOptions) error{}
	if endpoint != "" {
		opts = append(opts, config.WithBaseEndpoint(endpoint))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.UsePathStyle = true
		}
	})
	return &S3Store{client: client, bucket: bucket, prefix: prefix}, nil
}

func (s *S3Store) key(runID, path string) string {
	if s.prefix != "" {
		return s.prefix + "/" + runID + "/" + path
	}
	return runID + "/" + path
}

func (s *S3Store) Upload(ctx context.Context, runID, path string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(runID, path)),
		Body:   r,
	})
	if err != nil {
		return fmt.Errorf("s3 upload %s/%s: %w", runID, path, err)
	}
	return nil
}

func (s *S3Store) Download(ctx context.Context, runID, path string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(runID, path)),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download %s/%s: %w", runID, path, err)
	}
	return out.Body, nil
}

func (s *S3Store) List(ctx context.Context, runID string) ([]string, error) {
	prefix := s.key(runID, "")
	var paths []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("s3 list %s: %w", runID, err)
		}
		for _, obj := range page.Contents {
			rel := (*obj.Key)[len(prefix):]
			paths = append(paths, rel)
		}
	}
	return paths, nil
}
