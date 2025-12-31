package buildcontext

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/radiofrance/dib/pkg/logger"
)

// AzureUploader implements the FileUploader interface to upload files to Azure Blob Storage.
type AzureUploader struct {
	client    *azblob.Client
	container string
}

// NewAzureUploader creates a new instance of AzureUploader.
func NewAzureUploader(account, container string) (*AzureUploader, error) {
	if container == "" {
		return nil, fmt.Errorf("azure container name is required")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("invalid azure credentials: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", account)

	_, err = url.Parse(serviceURL)
	if account == "" || err != nil {
		return nil, fmt.Errorf("invalid azure storage service URL %q", serviceURL)
	}

	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}

	return &AzureUploader{
		client:    client,
		container: container,
	}, nil
}

// UploadFile uploads a file to the specified container and path in Azure Blob Storage.
func (u *AzureUploader) UploadFile(ctx context.Context, filePath, targetPath string) error {
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

	_, err = u.client.UploadFile(ctx, u.container, targetPath, file, nil)
	if err != nil {
		return fmt.Errorf("failed to upload file to Azure Blob Storage: %w", err)
	}

	return nil
}

// PresignedURL generates a SAS URL for accessing a blob in Azure Blob Storage.
func (u *AzureUploader) PresignedURL(ctx context.Context, targetPath string) (string, error) {
	now := time.Now().UTC()
	expiry := now.Add(1 * time.Hour)

	userDelegationCreds, err := u.client.ServiceClient().
		GetUserDelegationCredential(ctx, service.KeyInfo{
			Start:  to.Ptr(now.Format(time.RFC3339)),
			Expiry: to.Ptr(expiry.Format(time.RFC3339)),
		}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get user delegation key: %w", err)
	}

	perms := sas.BlobPermissions{Read: true}

	signValues := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now.Add(-15 * time.Minute), // Adjust for clock skew
		ExpiryTime:    expiry,
		Permissions:   perms.String(),
		ContainerName: u.container,
		BlobName:      targetPath,
	}

	sasQueryParams, err := signValues.SignWithUserDelegation(userDelegationCreds)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS token: %w", err)
	}

	baseURL, err := url.JoinPath(u.client.URL(), u.container, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS URL: %w", err)
	}

	encodedParams := sasQueryParams.Encode()
	presignedURL := strings.Join([]string{baseURL, encodedParams}, "?")

	return presignedURL, nil
}
