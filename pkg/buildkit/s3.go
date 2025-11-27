package buildkit

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

	"github.com/radiofrance/dib/internal/logger"
)

// S3Uploader is a FileUploader that uploads files to an AWS S3 bucket.
type S3Uploader struct {
	s3     *s3.Client
	bucket string
}

// NewS3Uploader creates a new instance of S3Uploader.
func NewS3Uploader(cfg aws.Config, bucket string) *S3Uploader {
	return &S3Uploader{
		s3:     s3.NewFromConfig(cfg),
		bucket: bucket,
	}
}

// UploadFile uploads a file to an AWS S3 bucket.
func (u S3Uploader) UploadFile(filePath string, targetPath string) error {
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
	fileInfo, _ := file.Stat()
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

	_, err = u.s3.PutObject(context.Background(), query)
	if err != nil {
		return fmt.Errorf("can't send S3 PUT request: %w", err)
	}

	return nil
}

// PresignedURL generates a presigned URL for accessing an object in the S3 bucket.
// The URL is valid for a limited time and allows temporary access to the specified object.
func (u S3Uploader) PresignedURL(targetPath string) (string, error) {
	presignClient := s3.NewPresignClient(u.s3)
	presignParams := &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(targetPath),
	}

	presignedURL, err := presignClient.PresignGetObject(context.Background(), presignParams, func(o *s3.PresignOptions) {
		o.Expires = 1 * time.Hour
	})
	if err != nil {
		return "", fmt.Errorf("can't generate presigned URL: %w", err)
	}

	return presignedURL.URL, nil
}
