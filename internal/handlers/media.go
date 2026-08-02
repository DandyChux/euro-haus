package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dandychux/euro-haus/internal/services"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// MediaFile represents a file in the storage
type MediaFile struct {
	Key          string    `json:"key"`
	URL          string    `json:"url"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
	Type         string    `json:"type"`
	Folder       string    `json:"folder"`
}

// ListMediaResponse contains the list of media files
type ListMediaResponse struct {
	Files []MediaFile `json:"files"`
	Total int         `json:"total"`
}

// UploadMediaResponse represents the upload response
type UploadMediaResponse struct {
	Success bool      `json:"success"`
	File    MediaFile `json:"file"`
	Message string    `json:"message"`
}

// DeleteMediaRequest represents the delete request
type DeleteMediaRequest struct {
	Key string `json:"key"`
}

// EventFolder represents a folder in the events directory
type EventFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListEventFoldersResponse contains the list of event folders
type ListEventFoldersResponse struct {
	Folders []EventFolder `json:"folders"`
	Total   int           `json:"total"`
}

// ListMedia returns all media files from the storage
func ListMedia(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get S3 client from storage service
	if services.S3Client == nil {
		log.Printf("S3 client not initialized")
		http.Error(w, "Storage service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Get bucket name from environment
	bucketName := os.Getenv("SPACES_BUCKET")
	if bucketName == "" {
		bucketName = "euro-haus" // fallback
	}

	// List objects in the bucket
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}

	result, err := services.S3Client.ListObjectsV2(context.TODO(), params)
	if err != nil {
		log.Printf("Failed to list objects: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// objectPaginator := s3.NewListObjectsV2Paginator(services.S3Client, params, func(o *s3.ListObjectsV2PaginatorOptions) {
	// 	if v := int32(*params.MaxKeys); v != 0 {
	// 		o.Limit = v
	// 	}
	// })

	// var i int
	// for objectPaginator.HasMorePages() {
	// 	i++

	// 	page, err := objectPaginator.NextPage(context.TODO())
	// 	if err != nil {
	// 		log.Fatalf("failed to get page %v, %v", i, err)
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}

	// 	for _, obj := range page.Contents {
	// 		if obj.Key == nil {
	// 			continue
	// 		}
	// 		fmt.Println("Object: ", *obj.Key)
	// 	}
	// }

	// Build Space URL
	spaceURL := fmt.Sprintf("https://%s.nyc3.cdn.digitaloceanspaces.com", bucketName)

	// Convert S3 objects to MediaFile format
	var files []MediaFile
	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}

		// Determine file type and folder
		key := *obj.Key
		fileType := getFileType(key)
		folder := getFolder(key)

		// Skip if it's a folder marker
		if strings.HasSuffix(key, "/") {
			continue
		}

		file := MediaFile{
			Key:          key,
			URL:          fmt.Sprintf("%s/%s", spaceURL, key),
			LastModified: *obj.LastModified,
			Size:         *obj.Size,
			Type:         fileType,
			Folder:       folder,
		}

		files = append(files, file)
	}

	// Return response
	response := ListMediaResponse{
		Files: files,
		Total: len(files),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UploadMedia handles file uploads to storage using the storage service
func UploadMedia(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length")

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file: %v", err)
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get folder from form data
	folder := r.FormValue("folder")

	// Get content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = getMimeType(header.Filename)
	}

	// Upload using the storage service with folder
	fileURL, err := services.UploadFile(file, header.Filename, contentType, folder)
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	// Extract key from URL for the response
	// The URL format is: https://bucket.region.digitaloceanspaces.com/key
	urlParts := strings.Split(fileURL, "/")
	key := strings.Join(urlParts[3:], "/") // Skip protocol, domain, and get the path

	// Get file size
	size := header.Size

	// Create response
	response := UploadMediaResponse{
		Success: true,
		File: MediaFile{
			Key:          key,
			URL:          fileURL,
			LastModified: time.Now(),
			Size:         size,
			Type:         getFileType(header.Filename),
			Folder:       getFolder(key),
		},
		Message: "File uploaded successfully",
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// DeleteMedia handles file deletion from storage
func DeleteMedia(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get S3 client from storage service
	if services.S3Client == nil {
		log.Printf("S3 client not initialized")
		http.Error(w, "Storage service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Parse request
	var req DeleteMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "File key is required", http.StatusBadRequest)
		return
	}

	// Get bucket name from environment
	bucketName := os.Getenv("SPACES_BUCKET")
	if bucketName == "" {
		bucketName = "euro-haus" // fallback
	}

	// Delete from S3
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(req.Key),
	}

	_, err := services.S3Client.DeleteObject(context.TODO(), deleteInput)
	if err != nil {
		log.Printf("Failed to delete file: %v", err)
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted file: %s", req.Key)

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "File deleted successfully",
		"key":     req.Key,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// ListEventFolders returns all subfolders under the events/ directory
func ListEventFolders(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get S3 client from storage service
	if services.S3Client == nil {
		log.Printf("S3 client not initialized")
		http.Error(w, "Storage service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Get bucket name from environment
	bucketName := os.Getenv("SPACES_BUCKET")
	if bucketName == "" {
		bucketName = "euro-haus" // fallback
	}

	// List objects with events/ prefix
	params := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucketName),
		Prefix:    aws.String("events/"),
		Delimiter: aws.String("/"),
	}

	result, err := services.S3Client.ListObjectsV2(context.TODO(), params)
	if err != nil {
		log.Printf("Failed to list event folders: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract folder names from CommonPrefixes
	var folders []EventFolder
	for _, prefix := range result.CommonPrefixes {
		if prefix.Prefix == nil {
			continue
		}

		// Remove "events/" prefix and trailing "/"
		folderPath := *prefix.Prefix
		folderName := strings.TrimPrefix(folderPath, "events/")
		folderName = strings.TrimSuffix(folderName, "/")

		if folderName != "" {
			folders = append(folders, EventFolder{
				Name: folderName,
				Path: folderPath,
			})
		}
	}

	// Return response
	response := ListEventFoldersResponse{
		Folders: folders,
		Total:   len(folders),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UploadEventGallery handles file uploads to event gallery folders
func UploadEventGallery(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length")

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max for videos
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the event slug from form data
	eventSlug := r.FormValue("eventSlug")
	if eventSlug == "" {
		http.Error(w, "Event slug is required", http.StatusBadRequest)
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file: %v", err)
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Construct the gallery folder path
	galleryFolder := fmt.Sprintf("events/%s/gallery/", eventSlug)

	// Get content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = getMimeType(header.Filename)
	}

	// Upload using the storage service with the gallery folder
	fileURL, err := services.UploadFile(file, header.Filename, contentType, galleryFolder)
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	// Extract key from URL for the response
	urlParts := strings.Split(fileURL, "/")
	key := strings.Join(urlParts[3:], "/")

	// Get file size
	size := header.Size

	// Create response
	response := UploadMediaResponse{
		Success: true,
		File: MediaFile{
			Key:          key,
			URL:          fileURL,
			LastModified: time.Now(),
			Size:         size,
			Type:         getFileType(header.Filename),
			Folder:       getFolder(key),
		},
		Message: "File uploaded to event gallery successfully",
	}

	log.Printf("Successfully uploaded file to event gallery: %s", key)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UploadEventGalleryBatch handles multiple file uploads to event gallery folders in a single request
func UploadEventGalleryBatch(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length")

	// Parse multipart form with larger max size for multiple files
	err := r.ParseMultipartForm(500 << 20) // 500MB max for batch uploads
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the event slug from form data
	eventSlug := r.FormValue("eventSlug")
	if eventSlug == "" {
		http.Error(w, "Event slug is required", http.StatusBadRequest)
		return
	}

	// Get all files from the "files" field
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	fmt.Printf("Processing batch upload of %d files for event: %s", len(files), eventSlug)

	// Construct the gallery folder path
	galleryFolder := fmt.Sprintf("events/%s/gallery/", eventSlug)

	// Upload all files and collect results
	uploadedFiles := make([]MediaFile, 0, len(files))
	errors := make([]string, 0)

	for _, fileHeader := range files {
		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			errMsg := fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err)
			log.Printf(errMsg)
			errors = append(errors, errMsg)
			continue
		}

		// Get content type
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = getMimeType(fileHeader.Filename)
		}

		// Upload using the storage service
		fileURL, err := services.UploadFile(file, fileHeader.Filename, contentType, galleryFolder)
		file.Close() // Close immediately after upload

		if err != nil {
			errMsg := fmt.Sprintf("Failed to upload file %s: %v", fileHeader.Filename, err)
			log.Printf(errMsg)
			errors = append(errors, errMsg)
			continue
		}

		// Extract key from URL
		urlParts := strings.Split(fileURL, "/")
		key := strings.Join(urlParts[3:], "/")

		// Add to successful uploads
		uploadedFiles = append(uploadedFiles, MediaFile{
			Key:          key,
			URL:          fileURL,
			LastModified: time.Now(),
			Size:         fileHeader.Size,
			Type:         getFileType(fileHeader.Filename),
			Folder:       getFolder(key),
		})

		fmt.Printf("Successfully uploaded file: %s", key)
	}

	// Create response with both successes and errors
	response := map[string]interface{}{
		"success":       len(uploadedFiles) > 0,
		"files":         uploadedFiles,
		"totalUploaded": len(uploadedFiles),
		"totalFailed":   len(errors),
		"errors":        errors,
		"message":       fmt.Sprintf("Uploaded %d of %d files successfully", len(uploadedFiles), len(files)),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Helper functions

// getFileType determines the file type from the filename
func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico":
		return "image"
	case ".mp4", ".webm", ".ogg", ".mov", ".avi":
		return "video"
	default:
		return "other"
	}
}

// getFolder extracts the folder from the file key
func getFolder(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "root"
}

// getMimeType returns the MIME type based on file extension
func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg":
		return "video/ogg"
	case ".mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}
