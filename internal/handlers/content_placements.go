package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dandychux/euro-haus/internal/services"

	"github.com/gorilla/mux"
)

// ContentPlacement represents a media placement on the site
type ContentPlacement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Page        string    `json:"page"`
	Type        string    `json:"type"` // "image", "video", or "text"
	MediaURL    string    `json:"media_url"`
	MediaKey    string    `json:"media_key,omitempty"`
	TextContent string    `json:"text_content,omitempty"`
	HTML        bool      `json:"html,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
}

// UpdateContentPlacementRequest represents the update request
type UpdateContentPlacementRequest struct {
	MediaURL    string `json:"media_url"`
	MediaKey    string `json:"media_key,omitempty"`
	TextContent string `json:"text_content,omitempty"`
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

// getPlacement retrieves a placement by id.
func getPlacement(ctx context.Context, id string) (*ContentPlacement, error) {
	db := services.GetDB()
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	row := db.WithContext(ctx).Raw(`
		SELECT id, name, description, page, type,
		       COALESCE(media_url, ''),
		       COALESCE(media_key, ''),
		       COALESCE(text_content, ''),
		       html, updated_at, COALESCE(updated_by, '')
		FROM content_placements
		WHERE id = ?
	`, id).Row()

	var p ContentPlacement
	err := row.Scan(
		&p.ID, &p.Name, &p.Description, &p.Page, &p.Type,
		&p.MediaURL, &p.MediaKey, &p.TextContent,
		&p.HTML, &p.UpdatedAt, &p.UpdatedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// savePlacement upserts a placement.
func savePlacement(ctx context.Context, p ContentPlacement) error {
	db := services.GetDB()
	if db == nil {
		return errors.New("database not initialized")
	}

	err := db.WithContext(ctx).Exec(`
		INSERT INTO content_placements
			(id, name, description, page, type, media_url, media_key, text_content, html, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name         = EXCLUDED.name,
			description  = EXCLUDED.description,
			page         = EXCLUDED.page,
			type         = EXCLUDED.type,
			media_url    = EXCLUDED.media_url,
			media_key    = EXCLUDED.media_key,
			text_content = EXCLUDED.text_content,
			html         = EXCLUDED.html,
			updated_by   = EXCLUDED.updated_by,
			updated_at   = NOW()
	`,
		p.ID, p.Name, p.Description, p.Page, p.Type,
		nullableString(p.MediaURL), nullableString(p.MediaKey),
		nullableString(p.TextContent), p.HTML, nullableString(p.UpdatedBy),
	).Error
	return err
}

// getAllPlacements returns every placement.
func getAllPlacements(ctx context.Context) ([]ContentPlacement, error) {
	db := services.GetDB()
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := db.WithContext(ctx).Raw(`
		SELECT id, name, description, page, type,
		       COALESCE(media_url, ''),
		       COALESCE(media_key, ''),
		       COALESCE(text_content, ''),
		       html, updated_at, COALESCE(updated_by, '')
		FROM content_placements
		ORDER BY page, name
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ContentPlacement, 0)
	for rows.Next() {
		var p ContentPlacement
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Page, &p.Type,
			&p.MediaURL, &p.MediaKey, &p.TextContent,
			&p.HTML, &p.UpdatedAt, &p.UpdatedBy,
		); err != nil {
			log.Printf("scan placement: %v", err)
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetAllContentPlacements returns all content placements
func GetAllContentPlacements(w http.ResponseWriter, r *http.Request) {

	// Get all placements
	placements, err := getAllPlacements(r.Context())
	if err != nil {
		log.Printf("Error fetching placements from DB: %v", err)
		http.Error(w, "Failed to fetch placements", http.StatusInternalServerError)
		return
	}

	// Group by page for better organization
	groupedPlacements := make(map[string][]ContentPlacement)
	for _, p := range placements {
		groupedPlacements[p.Page] = append(groupedPlacements[p.Page], p)
	}

	response := map[string]interface{}{
		"placements": groupedPlacements,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetContentPlacement returns a single content placement by ID
func GetContentPlacement(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	p, err := getPlacement(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
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
	placement, err := getPlacement(r.Context(), id)
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
	if err := savePlacement(r.Context(), *placement); err != nil {
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
	existing, _ := getPlacement(r.Context(), placement.ID)
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
	if err := savePlacement(r.Context(), placement); err != nil {
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

	// Get all Placements
	placements, err := getAllPlacements(r.Context())
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
