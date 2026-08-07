package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// SetupCORS configures CORS middleware with appropriate settings
func SetupCORS(allowedOrigins []string) *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"Stripe-Signature",
		},
		ExposedHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
}
