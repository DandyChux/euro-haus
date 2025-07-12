package handlers

import (
	"context"
	"euro-haus-api/services"
	"log"
	"strings"
)

// InitEventListeners initializes listeners for all active events
func InitEventListeners() {
	// Get Redis client from service
	rdb := services.GetRedisClient()
	if rdb == nil {
		log.Println("Redis not available, skipping event listeners initialization")
		return
	}

	ctx := context.Background()

	// Scan for all event attendee sets
	var cursor uint64
	pattern := "event:*:attendees"

	log.Println("Initializing event listeners...")

	for {
		var keys []string
		var err error
		keys, cursor, err = rdb.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			log.Printf("Error scanning Redis for event keys: %v", err)
			break
		}

		// Extract event IDs and start listeners
		for _, key := range keys {
			// Extract event ID from key (format: event:{id}:attendees)
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				eventID := parts[1]
				log.Printf("Starting listener for event: %s", eventID)
				StartEventUpdatesListener(eventID)
			}
		}

		if cursor == 0 {
			break
		}
	}

	log.Println("Event listeners initialization complete")
}
