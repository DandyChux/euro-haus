package handlers

import (
	"context"
	"encoding/json"
	"euro-haus-api/services"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

// ContentPlacement represents a media placement on the site
type ContentPlacement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Page        string    `json:"page"`
	Type        string    `json:"type"` // "image", "video", or "text"
	MediaURL    string    `json:"mediaUrl"`
	MediaKey    string    `json:"mediaKey,omitempty"`
	TextContent string    `json:"textContent,omitempty"` // New field for text content
	HTML        bool      `json:"html,omitempty"`        // Whether text should be rendered as HTML
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   string    `json:"updatedBy,omitempty"`
}

// UpdateContentPlacementRequest represents the update request
type UpdateContentPlacementRequest struct {
	MediaURL    string `json:"mediaUrl"`
	MediaKey    string `json:"mediaKey,omitempty"`
	TextContent string `json:"textContent,omitempty"`
}

const (
	placementKeyPrefix = "placement:"
	placementListKey   = "placements:all"
)

var (
	// In-memory fallback storage
	memoryPlacements     = make(map[string]ContentPlacement)
	memoryPlacementMutex sync.RWMutex
	ctx                  = context.Background()
)

// getPlacementFromRedis retrieves a placement from Redis
func getPlacementFromRedis(id string) (*ContentPlacement, error) {
	redisClient := services.GetRedisClient()
	if redisClient == nil {
		// Fallback to in-memory storage
		memoryPlacementMutex.RLock()
		defer memoryPlacementMutex.RUnlock()
		if placement, exists := memoryPlacements[id]; exists {
			return &placement, nil
		}
		return nil, nil
	}

	data, err := redisClient.Get(ctx, placementKeyPrefix+id).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var placement ContentPlacement
	if err := json.Unmarshal([]byte(data), &placement); err != nil {
		return nil, err
	}

	return &placement, nil
}

// savePlacementToRedis saves a placement to Redis
func savePlacementToRedis(placement ContentPlacement) error {
	redisClient := services.GetRedisClient()
	if redisClient == nil {
		// Fallback to in-memory storage
		memoryPlacementMutex.Lock()
		defer memoryPlacementMutex.Unlock()
		memoryPlacements[placement.ID] = placement
		return nil
	}

	data, err := json.Marshal(placement)
	if err != nil {
		return err
	}

	// Save the placement
	if err := redisClient.Set(ctx, placementKeyPrefix+placement.ID, data, 0).Err(); err != nil {
		return err
	}

	// Add to the list of all placements
	if err := redisClient.SAdd(ctx, placementListKey, placement.ID).Err(); err != nil {
		return err
	}

	return nil
}

// getAllPlacementsFromRedis retrieves all placements from Redis
func getAllPlacementsFromRedis() ([]ContentPlacement, error) {
	redisClient := services.GetRedisClient()
	if redisClient == nil {
		// Fallback to in-memory storage
		memoryPlacementMutex.RLock()
		defer memoryPlacementMutex.RUnlock()
		placements := make([]ContentPlacement, 0, len(memoryPlacements))
		for _, p := range memoryPlacements {
			placements = append(placements, p)
		}
		return placements, nil
	}

	// Get all placement IDs
	ids, err := redisClient.SMembers(ctx, placementListKey).Result()
	if err != nil {
		return nil, err
	}

	placements := make([]ContentPlacement, 0, len(ids))
	for _, id := range ids {
		placement, err := getPlacementFromRedis(id)
		if err != nil {
			log.Printf("Error fetching placement %s: %v", id, err)
			continue
		}
		if placement != nil {
			placements = append(placements, *placement)
		}
	}

	return placements, nil
}

// GetAllContentPlacements returns all content placements
func GetAllContentPlacements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get all placements from Redis
	placements, err := getAllPlacementsFromRedis()
	if err != nil {
		log.Printf("Error fetching placements from Redis: %v", err)
		http.Error(w, "Failed to fetch placements", http.StatusInternalServerError)
		return
	}

	// Group by page for better organization
	groupedPlacements := make(map[string][]ContentPlacement)
	for _, p := range placements {
		groupedPlacements[p.Page] = append(groupedPlacements[p.Page], p)
	}

	response := map[string]interface{}{
		"placements": placements,
		"grouped":    groupedPlacements,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetContentPlacement returns a single content placement by ID
func GetContentPlacement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get ID from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Get from Redis
	placement, err := getPlacementFromRedis(id)
	if err != nil {
		log.Printf("Error fetching placement: %v", err)
		http.Error(w, "Failed to fetch placement", http.StatusInternalServerError)
		return
	}

	if placement == nil {
		http.Error(w, "Placement not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"placement": placement,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateContentPlacement updates a content placement
func UpdateContentPlacement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get ID from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Parse request body
	var req UpdateContentPlacementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing placement
	placement, err := getPlacementFromRedis(id)
	if err != nil {
		log.Printf("Error fetching placement: %v", err)
		http.Error(w, "Failed to fetch placement", http.StatusInternalServerError)
		return
	}

	if placement == nil {
		http.Error(w, "Placement not found", http.StatusNotFound)
		return
	}

	// Update placement based on type
	if placement.Type == "text" {
		// Update text content
		if req.TextContent != "" {
			placement.TextContent = req.TextContent
		}
	} else {
		// Update media URL and key for media types
		if req.MediaURL != "" {
			placement.MediaURL = req.MediaURL
		}
		if req.MediaKey != "" {
			placement.MediaKey = req.MediaKey
		}
	}

	placement.UpdatedAt = time.Now()
	placement.UpdatedBy = "admin"

	// Save to Redis
	if err := savePlacementToRedis(*placement); err != nil {
		log.Printf("Error saving placement: %v", err)
		http.Error(w, "Failed to update placement", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"placement": placement,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("Updated content placement %s with new media: %s", id, req.MediaURL)
}

// RegisterContentPlacement registers a new dynamic placement
func RegisterContentPlacement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var placement ContentPlacement
	if err := json.NewDecoder(r.Body).Decode(&placement); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if placement already exists
	existing, _ := getPlacementFromRedis(placement.ID)
	if existing != nil {
		// Update the timestamp but don't override media URL if it's been customized
		if existing.MediaURL != placement.MediaURL && existing.MediaKey != "" {
			// This placement has been customized, don't override
			placement.MediaURL = existing.MediaURL
			placement.MediaKey = existing.MediaKey
		}
	}

	// Set timestamp
	placement.UpdatedAt = time.Now()

	// Save to Redis
	if err := savePlacementToRedis(placement); err != nil {
		log.Printf("Error saving placement: %v", err)
		http.Error(w, "Failed to register placement", http.StatusInternalServerError)
		return
	}

	log.Printf("Registered dynamic placement: %s on page %s", placement.Name, placement.Page)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetContentPlacementsByMedia returns all placements using a specific media key
func GetContentPlacementsByMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get media key from URL
	vars := mux.Vars(r)
	mediaKey := vars["key"]

	// Get all placements
	placements, err := getAllPlacementsFromRedis()
	if err != nil {
		log.Printf("Error fetching placements: %v", err)
		http.Error(w, "Failed to fetch placements", http.StatusInternalServerError)
		return
	}

	// Filter by media key
	var results []ContentPlacement
	for _, placement := range placements {
		if placement.MediaKey == mediaKey {
			results = append(results, placement)
		}
	}

	response := map[string]interface{}{
		"placements": results,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Legacy endpoints for backward compatibility
func GetDynamicContentPlacements(w http.ResponseWriter, r *http.Request) {
	GetAllContentPlacements(w, r)
}

func GetContentPlacements(w http.ResponseWriter, r *http.Request) {
	GetAllContentPlacements(w, r)
}
