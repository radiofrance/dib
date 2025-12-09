package buildcontext

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/radiofrance/dib/pkg/logger"
)

// S3Uploader implements the FileUploader interface to upload files to any S3-compatible bucket.
type S3Uploader struct {
	s3     *s3.Client
	bucket string
}

func NewS3Uploader(cfg aws.Config, bucket string) *S3Uploader {
	return &S3Uploader{
		s3:     s3.NewFromConfig(cfg),
		bucket: bucket,
	}
}

func (u *S3Uploader) UploadFile(ctx context.Context, filePath, targetPath string) error {
	file, err := os.Open(filePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("can't open file %s: %w", filePath, err)
	}

	defer func() {
		err := file.Close()
		if err != nil {
			logger.Errorf("can't close file %s: %v", filePath, err)
		}
	}()

	// Get file size and read the file content into a buffer
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("can't get file info for file %s: %w", filePath, err)
	}

	size := fileInfo.Size()
	buffer := make([]byte, size)

	_, err = file.Read(buffer)
	if err != nil {
		return fmt.Errorf("can't read file %s: %w", filePath, err)
	}

	query := &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(targetPath),
		ACL:           types.ObjectCannedACLPrivate,
		Body:          bytes.NewReader(buffer),
		ContentLength: &size,
		ContentType:   aws.String(http.DetectContentType(buffer)),
	}

	_, err = u.s3.PutObject(ctx, query)
	if err != nil {
		return fmt.Errorf("can't send S3 PUT request: %w", err)
	}

	return nil
}

// PresignedURL generates a presigned URL for accessing an object in any S3 bucket.
// The URL is valid for a limited time and allows temporary access to the specified object.
func (u *S3Uploader) PresignedURL(ctx context.Context, targetPath string) (string, error) {
	presignClient := s3.NewPresignClient(u.s3)
	presignParams := &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(targetPath),
	}

	presignedURL, err := presignClient.PresignGetObject(ctx, presignParams,
		func(o *s3.PresignOptions) {
			o.Expires = 1 * time.Hour
		})
	if err != nil {
		return "", fmt.Errorf("can't generate presigned URL: %w", err)
	}

	return presignedURL.URL, nil
}
