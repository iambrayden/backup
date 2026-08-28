package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Credentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type Client struct {
	s3     *s3.Client
	bucket string
}

func New(endpoint, region, bucket, accessKeyID, secretAccessKey string, pathStyle bool) *Client {
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, reg string, opts ...any) (aws.Endpoint, error) {
				if endpoint != "" {
					return aws.Endpoint{URL: endpoint, HostnameImmutable: true}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = pathStyle
	})
	return &Client{s3: client, bucket: bucket}
}

func NewFromTarget(endpoint, region, bucket string, credsEnc []byte, masterKey []byte, pathStyle bool) (*Client, error) {
	creds, err := DecryptCredentials(masterKey, credsEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	return New(endpoint, region, bucket, creds.AccessKeyID, creds.SecretAccessKey, pathStyle), nil
}

func (c *Client) PutObject(ctx context.Context, key string, data []byte) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (c *Client) HeadObject(ctx context.Context, key string) (bool, error) {
	_, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *s3types.NotFound
		if ok := isNotFound(err, &nf); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isNotFound(err error, _ any) bool {
	return err != nil && (contains(err.Error(), "NoSuchKey") || contains(err.Error(), "NotFound") || contains(err.Error(), "404"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (c *Client) PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error) {
	presigner := s3.NewPresignClient(c.s3)
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil && !containsStr(err.Error(), "BucketAlreadyOwnedByYou") && !containsStr(err.Error(), "BucketAlreadyExists") {
		return err
	}
	return nil
}

func EncryptCredentials(masterKey []byte, creds Credentials) ([]byte, error) {
	plain, err := json.Marshal(creds)
	if err != nil {
		return nil, err
	}
	return encryptBytes(masterKey, plain)
}

func DecryptCredentials(masterKey, blob []byte) (*Credentials, error) {
	plain, err := decryptBytes(masterKey, blob)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
