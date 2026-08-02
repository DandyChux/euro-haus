package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/dandychux/euro-haus/internal/services"
	"github.com/jackc/pgx/v5"
)

func InitEventListeners() {
	pool := services.GetDB()
	if pool == nil {
		log.Println("Postgres not available, skipping event listener init")
		return
	}

	dsn := services.GetDatabaseDSN()
	if dsn == "" {
		log.Println("Postgres DSN not available, skipping event listener init")
		return
	}

	go runEventListener(dsn)
}

func runEventListener(connString string) {
	ctx := context.Background()
	for {
		conn, err := pgx.Connect(ctx, connString)
		if err != nil {
			log.Printf("Failed to connect for event listener: %v. Retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if _, err := conn.Exec(ctx, `LISTEN event_updates`); err != nil {
			log.Printf("LISTEN failed: %v", err)
			conn.Close(ctx)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Postgres LISTEN event_updates initialized successfully")

		for {
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				log.Printf("Error waiting for notification: %v", err)
				break
			}

			// Split the channel data "event_id|JSON_payload"
			parts := strings.SplitN(notification.Payload, "|", 2)
			if len(parts) == 2 {
				eventID := parts[0]
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(parts[1]), &data); err == nil {
					dispatchToEventSockets(eventID, data)
				} else {
					log.Printf("Error unmarshaling event broadcast data: %v", err)
				}
			}
		}
		conn.Close(ctx)
		time.Sleep(2 * time.Second)
	}
}

// dispatchToEventSockets sends a message to every websocket connection
// currently subscribed to eventID. Wire this up to your existing
// `eventConnections` map in event.go.
func dispatchToEventSockets(eventID string, msg map[string]interface{}) {
	eventConnectionsMutex.RLock()
	defer eventConnectionsMutex.RUnlock()

	for conn, evID := range eventConnections {
		if evID != eventID {
			continue
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("ws write failed: %v", err)
		}
	}
}
