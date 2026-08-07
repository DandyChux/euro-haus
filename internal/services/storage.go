package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// S3Client is the client for interacting with S3-compatible storage
var S3Client *s3.Client

// InitS3Client initializes the S3 client for DigitalOcean Spaces
func InitS3Client() {
	accessKeyID := strings.TrimSpace(os.Getenv("SPACES_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("SPACES_SECRET_KEY"))
	region := strings.TrimSpace(os.Getenv("SPACES_REGION"))
	endpoint := strings.TrimSpace(os.Getenv("SPACES_ENDPOINT"))
	bucket := strings.TrimSpace(os.Getenv("SPACES_BUCKET"))

	switch {
		case accessKeyID == "":
			log.Fatal("SPACES_ACCESS_KEY is required")
		case secretKey == "":
			log.Fatal("SPACES_SECRET_KEY is required")
		case region == "":
			log.Fatal("SPACES_REGION is required")
		case endpoint == "":
			log.Fatal("SPACES_ENDPOINT is required")
		case bucket == "":
			log.Fatal("SPACES_BUCKET is required")
	}

	baseEndpoint := endpoint
	if !strings.HasPrefix(baseEndpoint, "http://") &&
		!strings.HasPrefix(baseEndpoint, "https://") {
		baseEndpoint = "https://" + baseEndpoint
	}

	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretKey,
			"",
		),
		Region:       region,
		BaseEndpoint: aws.String(baseEndpoint),
	}

	S3Client = s3.NewFromConfig(*s3Config)
}

// UploadFile uploads a file to DigitalOcean Spaces
func UploadFile(fileData io.Reader, filename string, contentType string, folder string) (string, error) {
	if S3Client == nil {
		return "", fmt.Errorf("Spaces client is not initialized")
	}

	// Generate a unique filename
	ext := filepath.Ext(filename)
	basename := strings.TrimSuffix(filepath.Base(filename), ext)
	safeBasename := strings.ReplaceAll(basename, " ", "-")
	safeBasename = strings.ToLower(safeBasename)

	uniqueFilename := fmt.Sprintf("%s-%s%s", safeBasename, uuid.New().String()[:8], ext)

	// Read file data into buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, fileData); err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// If content type is not provided, detect it
	if contentType == "" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	// Use provided folder or default to date-based structure
	if folder == "" {
		// Default folder structure: organize by date: 2023/05/
		currentTime := time.Now()
		folder = fmt.Sprintf("%d/%02d", currentTime.Year(), currentTime.Month())
	}

	// Ensure folder doesn't have leading/trailing slashes and add trailing slash if needed
	folder = strings.Trim(folder, "/")
	if folder != "" {
		folder = folder + "/"
	}

	key := folder + uniqueFilename

	bucket := strings.TrimSpace(os.Getenv("SPACES_BUCKET"))
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("SPACES_ENDPOINT")), "/")

	if bucket == "" || endpoint == "" {
		return "", fmt.Errorf("Spaces bucket and endpoint are required")
	}

	// Upload to DigitalOcean Spaces
	_, err := S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: aws.Int64(int64(buf.Len())),
		ContentType:   aws.String(contentType),
		ACL:           types.ObjectCannedACLPublicRead,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Return the public URL
	fileURL := fmt.Sprintf(
		"https://%s.%s/%s",
		bucket,
		strings.TrimPrefix(endpoint, "https://"),
		key,
	)

	return fileURL, nil
}

// UploadJSON uploads JSON data to DigitalOcean Spaces
func UploadJSON(data interface{}, filename string, folder string) (string, error) {
	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Generate unique filename if not provided
	if filename == "" {
		filename = fmt.Sprintf("data-%s.json", uuid.New().String()[:8])
	} else if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	// Use provided folder or default
	if folder == "" {
		folder = "metadata"
	}

	// Ensure folder doesn't have leading/trailing slashes
	folder = strings.Trim(folder, "/")
	if folder != "" {
		folder = folder + "/"
	}

	key := folder + filename

	// Upload to DigitalOcean Spaces
	_, err = S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(os.Getenv("SPACES_BUCKET")),
		Key:           aws.String(key),
		Body:          bytes.NewReader(jsonData),
		ContentLength: aws.Int64(int64(len(jsonData))),
		ContentType:   aws.String("application/json"),
		ACL:           types.ObjectCannedACLPublicRead,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload JSON: %w", err)
	}

	// Return the public URL
	jsonURL := fmt.Sprintf("https://%s.%s/%s",
		os.Getenv("SPACES_BUCKET"),
		os.Getenv("SPACES_ENDPOINT"),
		key,
	)

	return jsonURL, nil
}
